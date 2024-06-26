// Package cmd is the entrypoint to cli
package cmd

import (
	"fmt"

	"github.com/brevdev/brev-cli/pkg/auth"
	"github.com/brevdev/brev-cli/pkg/cmd/delete"
	"github.com/brevdev/brev-cli/pkg/cmd/hello"
	"github.com/brevdev/brev-cli/pkg/cmd/login"
	"github.com/brevdev/brev-cli/pkg/cmd/logout"
	"github.com/brevdev/brev-cli/pkg/cmd/ls"
	"github.com/brevdev/brev-cli/pkg/cmd/portforward"
	"github.com/brevdev/brev-cli/pkg/cmd/refresh"
	"github.com/brevdev/brev-cli/pkg/cmd/set"
	"github.com/brevdev/brev-cli/pkg/cmd/sshkeys"
	"github.com/brevdev/brev-cli/pkg/cmd/start"
	"github.com/brevdev/brev-cli/pkg/cmd/stop"
	"github.com/brevdev/brev-cli/pkg/cmd/upgrade"
	"github.com/brevdev/brev-cli/pkg/config"
	"github.com/brevdev/brev-cli/pkg/featureflag"
	"github.com/brevdev/brev-cli/pkg/files"
	"github.com/brevdev/brev-cli/pkg/remoteversion"
	"github.com/brevdev/brev-cli/pkg/store"
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	breverrors "github.com/brevdev/brev-cli/pkg/errors"
)

var user string

func NewDefaultBrevCommand() *cobra.Command {
	cmd := NewBrevCommand()
	cmd.PersistentFlags().StringVar(&user, "user", "", "non root user to use for per user configuration of commands run as root")
	return cmd
}

func NewBrevCommand() *cobra.Command { //nolint:funlen,gocognit,gocyclo // define brev command
	// in io.Reader, out io.Writer, err io.Writer
	t := terminal.New()
	var printVersion bool

	conf := config.NewConstants()
	fs := files.AppFs
	authenticator := auth.Authenticator{
		Audience:           "https://brevdev.us.auth0.com/api/v2/",
		ClientID:           "JaqJRLEsdat5w7Tb0WqmTxzIeqwqepmk",
		DeviceCodeEndpoint: "https://brevdev.us.auth0.com/oauth/device/code",
		OauthTokenEndpoint: "https://brevdev.us.auth0.com/oauth/token",
	}
	// super annoying. this is needed to make the import stay
	_ = color.New(color.FgYellow, color.Bold).SprintFunc()

	fsStore := store.
		NewBasicStore().
		WithFileSystem(fs)
	loginAuth := auth.NewLoginAuth(fsStore, authenticator)
	noLoginAuth := auth.NewNoLoginAuth(fsStore, authenticator)

	loginCmdStore := fsStore.WithNoAuthHTTPClient(
		store.NewNoAuthHTTPClient(conf.GetBrevAPIURl()),
	).
		WithAuth(loginAuth, store.WithDebug(conf.GetDebugHTTP()))

	err := loginCmdStore.SetForbiddenStatusRetryHandler(func() error {
		_, err1 := loginAuth.GetAccessToken()
		if err1 != nil {
			return breverrors.WrapAndTrace(err1)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	noAuthCmdStore := fsStore.WithNoAuthHTTPClient(
		store.NewNoAuthHTTPClient(conf.GetBrevAPIURl()),
	)
	noLoginCmdStore := noAuthCmdStore.WithAuth(noLoginAuth)

	workspaceGroupID, err := fsStore.GetCurrentWorkspaceGroupID()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	if workspaceGroupID != "" {
		loginCmdStore.WithStaticHeader("X-Workspace-Group-ID", workspaceGroupID)
		noLoginCmdStore.WithStaticHeader("X-Workspace-Group-ID", workspaceGroupID)
	}

	cmds := &cobra.Command{
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "brev",
		Short:         "brev client for managing instances",
		Long: `
      brev client for managing instances

      Find more information at:
            https://brev.dev`,
		PostRun: func(cmd *cobra.Command, args []string) {
			shouldWe := hello.ShouldWeRunOnboarding(noLoginCmdStore)
			if shouldWe {
				user, err := loginCmdStore.GetCurrentUser()
				if err != nil {
					return
				}
				err = hello.CanWeOnboard(t, user, loginCmdStore)
				if err != nil {
					return
				}
			}
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			breverrors.GetDefaultErrorReporter().AddTag("command", cmd.Name())
			// version info gets in the way of the output for
			// configure-env-vars, since shells are going to eval it
			if featureflag.ShowVersionOnRun() && !printVersion && cmd.Name() != "configure-env-vars" {
				v, err := remoteversion.BuildCheckLatestVersionString(t, noLoginCmdStore)
				// todo this should not be fatal when it errors
				if err != nil {
					t.Errprint(err, "Failed to determine version")
					return breverrors.WrapAndTrace(err)
				}
				if v != "" {
					fmt.Println(v)
				}
			}
			if user != "" {
				_, err := noLoginCmdStore.WithUserID(user)
				if err != nil {
					return breverrors.WrapAndTrace(err)
				}
				_, err = loginCmdStore.WithUserID(user)
				if err != nil {
					return breverrors.WrapAndTrace(err)
				}
				_, err = fsStore.WithUserID(user)
				if err != nil {
					return breverrors.WrapAndTrace(err)
				}

			}
			home, err := fsStore.GetBrevHomePath()
			if err != nil {
				fmt.Printf("Warning: %v", err)
			}

			err = featureflag.LoadFeatureFlags(home)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if printVersion {
				v, err := remoteversion.BuildVersionString(t, noAuthCmdStore)
				if err != nil {
					t.Errprint(err, "Failed to determine version")
					return breverrors.WrapAndTrace(err)
				}
				t.Vprint(v)
				return nil
			} else {
				err := cmd.Usage()
				if err != nil {
					return breverrors.WrapAndTrace(err)
				}
				return nil
			}
		},
	}
	cobra.AddTemplateFunc("hasContextCommands", hasContextCommands)
	cobra.AddTemplateFunc("contextCommands", contextCommands)
	cobra.AddTemplateFunc("hasSSHCommands", hasSSHCommands)
	cobra.AddTemplateFunc("sshCommands", sshCommands)
	cobra.AddTemplateFunc("hasWorkspaceCommands", hasWorkspaceCommands)
	cobra.AddTemplateFunc("workspaceCommands", workspaceCommands)
	cobra.AddTemplateFunc("hasHousekeepingCommands", hasHousekeepingCommands)
	cobra.AddTemplateFunc("hasDebugCommands", hasDebugCommands)
	cobra.AddTemplateFunc("debugCommands", debugCommands)
	cobra.AddTemplateFunc("printCautiousMetaCmdMessage", printCautiousMetaCmdMessage)
	cobra.AddTemplateFunc("housekeepingCommands", housekeepingCommands)
	cobra.AddTemplateFunc("hasQuickstartCommands", hasQuickstartCommands)
	cobra.AddTemplateFunc("quickstartCommands", quickstartCommands)

	cmds.SetUsageTemplate(usageTemplate)

	cmds.PersistentFlags().BoolVar(&printVersion, "version", false, "Print version output")

	createCmdTree(cmds, t, loginCmdStore, noLoginCmdStore, loginAuth)

	return cmds
}

func createCmdTree(cmd *cobra.Command, t *terminal.Terminal, loginCmdStore *store.AuthHTTPStore, noLoginCmdStore *store.AuthHTTPStore, loginAuth *auth.LoginAuth) { //nolint:funlen,nolintlint // define brev command
	cmd.AddCommand(set.NewCmdSet(t, loginCmdStore, noLoginCmdStore))
	cmd.AddCommand(ls.NewCmdLs(t, loginCmdStore, noLoginCmdStore))
	cmd.AddCommand(portforward.NewCmdPortForwardSSH(loginCmdStore, t))
	cmd.AddCommand(login.NewCmdLogin(t, noLoginCmdStore, loginAuth))
	cmd.AddCommand(logout.NewCmdLogout(loginAuth, noLoginCmdStore))
	cmd.AddCommand(sshkeys.NewCmdSSHKeys(t, loginCmdStore))
	cmd.AddCommand(start.NewCmdStart(t, loginCmdStore, noLoginCmdStore))
	cmd.AddCommand(stop.NewCmdStop(t, loginCmdStore, noLoginCmdStore))
	cmd.AddCommand(delete.NewCmdDelete(t, loginCmdStore, noLoginCmdStore))
	cmd.AddCommand(refresh.NewCmdRefresh(t, loginCmdStore))
	cmd.AddCommand(hello.NewCmdHello(t, noLoginCmdStore))
	cmd.AddCommand(upgrade.NewCmdUpgrade(t, loginCmdStore))
}

func hasQuickstartCommands(cmd *cobra.Command) bool {
	return len(quickstartCommands(cmd)) > 0
}

func hasHousekeepingCommands(cmd *cobra.Command) bool {
	return len(housekeepingCommands(cmd)) > 0
}

func hasDebugCommands(cmd *cobra.Command) bool {
	return len(debugCommands(cmd)) > 0
}

func printCautiousMetaCmdMessage() string {
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	return yellow("(we're actively working on getting rid of these commands)")
}

func hasSSHCommands(cmd *cobra.Command) bool {
	return len(sshCommands(cmd)) > 0
}

func hasWorkspaceCommands(cmd *cobra.Command) bool {
	return len(workspaceCommands(cmd)) > 0
}

func hasContextCommands(cmd *cobra.Command) bool {
	return len(contextCommands(cmd)) > 0
}

func quickstartCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isQuickstartCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func housekeepingCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isHousekeepingCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func debugCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isDebugCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func sshCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isSSHCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func workspaceCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isWorkspaceCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func contextCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isContextCommand(sub) {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func isQuickstartCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["quickstart"]
	return ok
}

func isHousekeepingCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["housekeeping"]
	return ok
}

func isDebugCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["debug"]
	return ok
}

func isSSHCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["ssh"]
	return ok
}

func isWorkspaceCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["workspace"]
	return ok
}

func isContextCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations["context"]
	return ok
}

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{- if hasWorkspaceCommands . }}

Instance Commands:
{{- range workspaceCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

{{- if hasSSHCommands . }}

{{- if hasContextCommands . }}

Context Commands:
{{- range contextCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

SSH Commands:
{{- range sshCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

{{- if hasQuickstartCommands . }}

Quickstart Commands:
{{- range quickstartCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

{{- if hasHousekeepingCommands . }}

Housekeeping Commands:
{{- range housekeepingCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

{{- if hasDebugCommands . }}

Debug Commands:
{{- range debugCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}{{- end}}

{{- end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var (
	_ store.Auth     = auth.LoginAuth{}
	_ store.Auth     = auth.NoLoginAuth{}
	_ auth.AuthStore = store.FileStore{}
)
