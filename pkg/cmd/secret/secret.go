// Package secret lets you add secrests. Language Go makes you write extra.
package secret

import (
	"encoding/json"
	"fmt"

	"github.com/brevdev/brev-cli/pkg/brevapi"
	"github.com/brevdev/brev-cli/pkg/cmdcontext"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/files"
	"github.com/brevdev/brev-cli/pkg/store"
	"github.com/brevdev/brev-cli/pkg/terminal"

	"github.com/spf13/cobra"
)

func GetAllWorkspaces(orgID string) ([]brevapi.Workspace, error) {
	client, err := brevapi.NewCommandClient()
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}
	wss, err := client.GetWorkspaces(orgID)
	if err != nil {
		return nil, breverrors.WrapAndTrace(err)
	}

	return wss, nil
}

type SecretStore interface {
	CreateSecret(req store.CreateSecretRequest) (*store.CreateSecretRequest, error)
}

func NewCmdSecret(secretStore SecretStore, t *terminal.Terminal) *cobra.Command {
	var envtype string
	var name string
	var value string
	var path string
	var scope string

	cmd := &cobra.Command{
		Annotations: map[string]string{"housekeeping": ""},
		Use:         "secret",
		Short:       "Add a secret/environment variable",
		Long:        "Add a secret/environment variable to your workspace, all workspaces in an org, or all of your workspaces",
		Example: `
  brev secret --name naaamme --value vaaalluueee --type [file, variable] --file-path --scope personal
  brev secret --name SERVER_URL --value https://brev.sh --type variable --scope personal
  brev secret --name AWS_KEY --value ... --type file --file-path --scope personal
		`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdcontext.InvokeParentPersistentPreRun(cmd, args)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}

			return nil
		},
		// Args:      cobra.MinimumNArgs(0),
		// ValidArgs: []string{"orgs", "workspaces"},
		RunE: func(cmd *cobra.Command, args []string) error {
			addSecret(secretStore, t, envtype, name, value, path, scope)
			return nil
		},
	}

	cmd.Flags().StringVarP(&envtype, "type", "t", "", "type of secret (env var or file)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "name of environment variable or secret file")
	cmd.Flags().StringVarP(&value, "value", "v", "", "value of environment variable or secret file")
	cmd.Flags().StringVarP(&path, "file-path", "p", "", "file path (if secret file)")
	cmd.Flags().StringVarP(&scope, "scope", "s", "", "scope for env var (org or private)")

	err := cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"file", "variable"}, cobra.ShellCompDirectiveNoSpace
	})
	if err != nil {
		t.Errprint(err, "cli err")
	}

	err = cmd.RegisterFlagCompletionFunc("scope", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"org", "private"}, cobra.ShellCompDirectiveNoSpace
	})
	if err != nil {
		t.Errprint(err, "cli err")
	}

	return cmd
}

func addSecret(secretStore SecretStore, t *terminal.Terminal, envtype string, name string, value string, path string, scope string) error {
	if name == "" || envtype == "" || value == "" || path == "" {
		t.Vprintf(t.Yellow("\nSome flags omitted, running interactive mode!\n"))
	}

	if name == "" {
		name = brevapi.PromptGetInput(brevapi.PromptContent{
			Label:    "Environment variable/secret name: ",
			ErrorMsg: "error",
		})
	}

	if envtype == "" {
		envtype = brevapi.PromptSelectInput(brevapi.PromptSelectContent{
			Label:    "Type of variable: ",
			ErrorMsg: "error",
			Items:    []string{"file", "variable"},
		})
	}

	if value == "" {
		value = brevapi.PromptGetInput(brevapi.PromptContent{
			Label:    "Environment variable/secret value: ",
			ErrorMsg: "error",
		})
	}

	if path == "" && envtype == "file" {
		path = brevapi.PromptGetInput(brevapi.PromptContent{
			Label:    "Path for the file: ",
			ErrorMsg: "error",
			Default:  "/home/brev/workspace/secret.txt",
		})
	}

	if scope == "" {
		scope = brevapi.PromptSelectInput(brevapi.PromptSelectContent{
			Label:    "Scope: ",
			ErrorMsg: "error",
			Items:    []string{"org", "user"},
		})
	}

	if envtype == "file" {
		t.Vprintf("brev secret --name %s --value %s --type %s --file-path %s --scope %s\n", name, value, envtype, path, scope)
	} else {
		t.Vprintf("brev secret --name %s --value %s --type %s --scope %s\n", name, value, envtype, scope)
	}


	s := t.NewSpinner()
	s.Suffix = "  encrypting and saving secret var"
	s.Start()

	iScope := store.Org
	var hierarchyId string
	if scope == "user" {
		iScope = store.User
		// get user id
		me, err := brevapi.GetMe()
		if err != nil {
			return err
		}
		hierarchyId = me.ID
	} else {
		// get org id 
		activeorg, err := brevapi.GetActiveOrgContext(files.AppFs)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		hierarchyId = activeorg.ID
	}
	
	var configDest store.DestConfig
	iType := store.File
	if envtype == "variable" {
		iType = store.EnvVariable
		configDest = store.DestConfig{
			Name: name,
		}
	} else {
		configDest = store.DestConfig{
			Path: path,
		} 
	}

	// NOTE: hieararchyID needs to be the org ID user ID

	b := store.CreateSecretRequest{
		Name:          name,
		HierarchyType: iScope,
		HierarchyId:   hierarchyId,
		Src: store.SecretReqSrc{
			Type: store.KeyValue,
			Config: store.SrcConfig{
				Value: value,
			},
		},
		Dest: store.SecretReqDest{
			Type: iType,
			Config: configDest,
		},
	}
	asstring, _ := json.MarshalIndent(b, "", "\t")
	fmt.Print(string(asstring))
	secret, err := secretStore.CreateSecret(b)
	if err != nil {
		s.Stop()
		t.Vprintf(t.Red(err.Error()))
		return err
	}
	t.Vprintf(secret.Name)
	s.Suffix = "  environment secret added"
	s.Stop()

	t.Vprintf(t.Green("\nEnvironment %s added\n", iType) + t.Yellow("\tNote: It might take up to 2 minutes to load into your environment."))

	return nil
}