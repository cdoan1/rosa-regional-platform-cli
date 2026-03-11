package clusteriam

import (
	"github.com/spf13/cobra"
)

// NewClusterIAMCommand creates the cluster-iam command
func NewClusterIAMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-iam",
		Short: "Manage cluster IAM resources (OIDC provider + roles)",
		Long: `Manage IAM resources for ROSA hosted clusters.

This command creates IAM OIDC providers and roles required for hosted control
plane clusters to interact with AWS services.

The resources are created via CloudFormation templates applied by the Lambda
function. The Lambda function must be bootstrapped first using 'rosactl bootstrap'.`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDescribeCommand())

	return cmd
}
