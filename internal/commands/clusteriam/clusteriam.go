package clusteriam

import (
	"github.com/spf13/cobra"
)

// NewClusterIAMCommand creates the cluster-iam command
func NewClusterIAMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-iam",
		Short: "Manage cluster IAM roles",
		Long: `Manage IAM roles for ROSA hosted clusters.

This command creates the IAM roles required for hosted control plane clusters
to interact with AWS services. Roles are created before the cluster exists;
OIDC federation is activated separately via 'rosactl oidc create' once the
cluster's issuer URL is known.

The resources are created via CloudFormation stacks (rosa-{cluster-name}-iam).`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDescribeCommand())

	return cmd
}
