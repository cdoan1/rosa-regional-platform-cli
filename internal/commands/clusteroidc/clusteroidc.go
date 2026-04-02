package clusteroidc

import (
	"github.com/spf13/cobra"
)

// NewClusterOIDCCommand creates the oidc command
func NewClusterOIDCCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oidc",
		Short: "Manage cluster OIDC provider resources",
		Long: `Manage the IAM OIDC provider for ROSA hosted clusters.

This command creates and deletes the AWS IAM OIDC provider required for
hosted control plane cluster service accounts to assume IAM roles via
Web Identity federation.

The OIDC provider is managed in its own CloudFormation stack (rosa-{cluster-name}-oidc),
separate from the IAM roles stack. Creating the OIDC provider also updates the IAM
roles stack trust policies with the real issuer domain.

Typical workflow:
  rosactl cluster-iam create my-cluster --region us-east-1
  rosactl cluster-vpc create my-cluster --region us-east-1
  # ... create cluster, obtain OIDC issuer URL ...
  rosactl oidc create my-cluster --oidc-issuer-url https://... --region us-east-1`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())

	return cmd
}
