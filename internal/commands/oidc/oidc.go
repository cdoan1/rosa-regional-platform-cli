package oidc

import "github.com/spf13/cobra"

func NewOIDCCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oidc",
		Short: "Manage OIDC issuers",
		Long:  "Create, list, delete, and manage OIDC issuers (S3-backed identity providers).",
	}
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDeleteCommand())
	return cmd
}
