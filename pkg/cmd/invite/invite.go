// Package invite is for inviting
package invite

import (
	"fmt"

	"github.com/brevdev/brev-cli/pkg/cmd/completions"
	"github.com/brevdev/brev-cli/pkg/cmdcontext"
	"github.com/brevdev/brev-cli/pkg/entity"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/store"
	"github.com/brevdev/brev-cli/pkg/terminal"

	"github.com/spf13/cobra"
)

type InviteStore interface {
	GetWorkspaces(organizationID string, options *store.GetWorkspacesOptions) ([]entity.Workspace, error)
	GetActiveOrganizationOrDefault() (*entity.Organization, error)
	GetCurrentUser() (*entity.User, error)
	GetUsers(queryParams map[string]string) ([]entity.User, error)
	GetWorkspace(workspaceID string) (*entity.Workspace, error)
	GetOrganizations(options *store.GetOrganizationsOptions) ([]entity.Organization, error)
	CreateInviteLink(organizationID string) (string, error)
}

func NewCmdInvite(t *terminal.Terminal, loginInviteStore InviteStore, noLoginInviteStore InviteStore) *cobra.Command {
	var showAll bool
	var org string

	cmd := &cobra.Command{
		Annotations: map[string]string{"housekeeping": ""},
		Use:         "invite",
		Short:       "Generate an invite link",
		Long:        "Get an invite link to your active org. Use the optional org flag to invite to a different org",
		Example: `
  brev invite
  brev ls --org <orgid>
		`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdcontext.InvokeParentPersistentPreRun(cmd, args)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}

			return nil
		},
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := RunInvite(t, loginInviteStore, args, org, showAll)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&org, "org", "o", "", "organization (will override active org)")
	err := cmd.RegisterFlagCompletionFunc("org", completions.GetOrgsNameCompletionHandler(noLoginInviteStore, t))
	if err != nil {
		t.Errprint(err, "cli err")
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "show all workspaces in org")

	return cmd
}

func RunInvite(t *terminal.Terminal, inviteStore InviteStore, args []string, orgflag string, showAll bool) error {
	var org *entity.Organization
	if orgflag != "" {
		orgs, err := inviteStore.GetOrganizations(&store.GetOrganizationsOptions{Name: orgflag})
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		if len(orgs) == 0 {
			return fmt.Errorf("no org found with name %s", orgflag)
		} else if len(orgs) > 1 {
			return fmt.Errorf("more than one org found with name %s", orgflag)
		}

		org = &orgs[0]
	} else {
		currOrg, err := inviteStore.GetActiveOrganizationOrDefault()
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		if currOrg == nil {
			return fmt.Errorf("no orgs exist")
		}
		org = currOrg
	}

	// TODO: generate the URL!!!
	token, err := inviteStore.CreateInviteLink(org.ID)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	t.Vprintf("Use the following invite link to invite someone to %s:", t.Green(org.Name))
	t.Vprintf("\n\n\t%s", t.Green("https://console.brev.dev/invite?token=%s\n", token))

	// err := ls.RunWorkspaces(org, showAll)
	// if err != nil {
	// 	return breverrors.WrapAndTrace(err)
	// }

	return nil
}