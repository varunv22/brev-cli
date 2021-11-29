package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/files"
	"github.com/pkg/browser"
	"github.com/spf13/afero"
)

type TempAuth struct{}

func (t TempAuth) GetAccessToken() (string, error) {
	return GetAccessToken()
}

type AuthStore interface {
	SaveAuthTokens(tokens AuthTokens) error
	GetAuthTokens() (*AuthTokens, error)
	DeleteAuthTokens() error
}

type OAuth interface {
	DoDeviceAuthFlow(onStateRetrieve func(url string, code string)) (*LoginTokens, error)
	GetNewAuthTokensWithRefresh(refreshToken string) (*AuthTokens, error)
}

type Auth struct {
	authStore AuthStore
	oauth     OAuth
}

func NewAuth(authStore AuthStore, oauth OAuth) *Auth {
	return &Auth{
		authStore: authStore,
		oauth:     oauth,
	}
}

// Gets fresh access token and prompts for login and saves to store
func (t Auth) GetFreshAccessTokenOrLogin() (string, error) {
	token, err := t.GetFreshAccessTokenOrNil()
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}
	if token == "" {
		lt, err := t.PromptForLogin()
		if err != nil {
			return "", breverrors.WrapAndTrace(err)
		}
		token = lt.accessToken
	}
	return token, nil
}

// Gets fresh access token or returns nil and saves to store
func (t Auth) GetFreshAccessTokenOrNil() (string, error) {
	tokens, err := t.getSavedTokensOrNil()
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}
	if tokens == nil {
		return "", nil
	}
	isAccessTokenExpired, err := t.isTokenExpired(tokens.accessToken)
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}
	if isAccessTokenExpired {
		tokens, err = t.getNewTokensWithRefreshOrNil(tokens.refreshToken)
		if tokens == nil {
			return "", nil
		}
		if err != nil {
			return "", breverrors.WrapAndTrace(err)
		}
	}
	return tokens.accessToken, nil
}

// Prompts for login and returns tokens, and saves to store
func (t Auth) PromptForLogin() (*LoginTokens, error) {
	reader := bufio.NewReader(os.Stdin) // TODO inject?
	fmt.Print(`You are currently logged out, would you like to log in? [y/n]: `)
	text, _ := reader.ReadString('\n')
	if strings.Compare(text, "y") != 1 {
		return nil, &breverrors.DeclineToLoginError{}
	}

	tokens, err := t.oauth.DoDeviceAuthFlow(
		func(url, code string) {
			fmt.Println("Your Device Confirmation Code is", code)

			err := browser.OpenURL(url)
			if err != nil {
				fmt.Println("please open: ", url)
			}

			fmt.Println("waiting for auth to complete")
		},
	)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err, "login error")
	}

	err = t.authStore.SaveAuthTokens(tokens.AuthTokens)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	fmt.Print("\n")
	fmt.Println("Successfully logged in.")

	return tokens, nil
}

func (t Auth) Logout() error {
	err := t.authStore.DeleteAuthTokens()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

type AuthTokens struct {
	accessToken  string
	refreshToken string
}

type LoginTokens struct {
	AuthTokens
	// idToken string
}

func (t Auth) getSavedTokensOrNil() (*AuthTokens, error) {
	tokens, err := t.authStore.GetAuthTokens()
	// TODO handle certain errors and return nil
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	return tokens, nil
}

// gets new access and refresh token or returns nil if refresh token expired, and updates store
func (t Auth) getNewTokensWithRefreshOrNil(refreshToken string) (*AuthTokens, error) {
	isRefreshTokenExpired, err := t.isTokenExpired(refreshToken)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	if isRefreshTokenExpired {
		return nil, nil
	}
	tokens, err := t.oauth.GetNewAuthTokensWithRefresh(refreshToken)
	// TODO handle if 403
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	err = t.authStore.SaveAuthTokens(*tokens)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	return tokens, nil
}

func (t Auth) isTokenExpired(_ string) (bool, error) {
	// TODO
	return false, nil
}

// #########################################################
const brevCredentialsFile = "credentials.json"

func GetAccessToken() (string, error) {
	oauthToken, err := GetToken()
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}

	return oauthToken.AccessToken, nil
}

func getBrevCredentialsFile() (*string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	brevCredentialsFile := home + "/" + files.GetBrevDirectory() + "/" + brevCredentialsFile
	return &brevCredentialsFile, nil
}

func WriteTokenToBrevConfigFile(token *Credentials) error {
	brevCredentialsFile, err := getBrevCredentialsFile()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = files.OverwriteJSON(*brevCredentialsFile, token)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	return nil
}

func GetTokenFromBrevConfigFile(fs afero.Fs) (*OauthToken, error) {
	brevCredentialsFile, err := getBrevCredentialsFile()
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	exists, err := afero.Exists(fs, *brevCredentialsFile)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	if !exists {
		return nil, &breverrors.CredentialsFileNotFound{}
	}

	var token OauthToken
	err = files.ReadJSON(fs, *brevCredentialsFile, &token)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	return &token, nil
}

func Login(prompt bool) (*string, error) {
	if prompt {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(`You are currently logged out, would you like to log in? [y/n]: `)
		text, _ := reader.ReadString('\n')
		if strings.Compare(text, "y") != 1 {
			return nil, &breverrors.DeclineToLoginError{}
		}
	}
	ctx := context.Background()

	// TODO env vars
	authenticator := Authenticator{
		Audience:           "https://brevdev.us.auth0.com/api/v2/",
		ClientID:           "JaqJRLEsdat5w7Tb0WqmTxzIeqwqepmk",
		DeviceCodeEndpoint: "https://brevdev.us.auth0.com/oauth/device/code",
		OauthTokenEndpoint: "https://brevdev.us.auth0.com/oauth/token",
	}
	state, err := authenticator.Start(ctx)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err, "could not start the authentication process")
	}

	// todo color library
	// fmt.Printf("Your Device Confirmation code is: %s\n\n", ansi.Bold(state.UserCode))
	// cli.renderer.Infof("%s to open the browser to log in or %s to quit...", ansi.Green("Press Enter"), ansi.Red("^C"))
	// fmt.Scanln()
	// TODO make this stand out! its important
	fmt.Println("Your Device Confirmation Code is", state.UserCode)

	err = browser.OpenURL(state.VerificationURI)

	if err != nil {
		fmt.Println("please open: ", state.VerificationURI)
	}

	fmt.Println("waiting for auth to complete")
	var res Result

	res, err = authenticator.Wait(ctx, state)

	if err != nil {
		return nil, breverrors.WrapAndTrace(err, "login error")
	}

	fmt.Print("\n")
	fmt.Println("Successfully logged in.")
	creds := &Credentials{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    int(res.ExpiresIn),
		IDToken:      res.IDToken,
	}
	// store the refresh token
	err = WriteTokenToBrevConfigFile(creds)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	// hydrate the cache
	// _, _, err = WriteCaches()
	// if err != nil {
	// 	return nil, breverrors.WrapAndTrace(err)
	// }

	return &creds.IDToken, nil
}

func Logout() error {
	brevCredentialsFile, err := getBrevCredentialsFile()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = files.DeleteFile(*brevCredentialsFile)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

// GetToken reads the previously-persisted token from the filesystem,
// returning nil for a token if it does not exist
func GetToken() (*OauthToken, error) {
	token, err := GetTokenFromBrevConfigFile(files.AppFs)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	if token == nil { // we have not logged in yet
		_, err = Login(true)
		if err != nil {
			return nil, breverrors.WrapAndTrace(err)
		}
		// now that we have logged in, the file should contain the token
		token, err = GetTokenFromBrevConfigFile(files.AppFs)
		if err != nil {
			return nil, breverrors.WrapAndTrace(err)
		}
	}
	return token, nil
}
