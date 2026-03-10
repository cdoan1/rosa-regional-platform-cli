package bootstrap

import (
	"github.com/spf13/cobra"
)

// NewBootstrapCommand creates the bootstrap command
func NewBootstrapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the Lambda function for cluster IAM management",
		Long: `Bootstrap creates the Lambda function infrastructure required for managing
cluster IAM resources. This is a one-time setup per AWS region/account.

The Lambda function is deployed as a container image and will be used to
apply CloudFormation templates for cluster IAM resources.`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newStatusCommand())

	return cmd
}
