package lambda

import (
	"github.com/spf13/cobra"
)

// NewLambdaCommand creates the lambda command
func NewLambdaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lambda",
		Short: "Manage Lambda functions for cluster resource queries",
		Long: `Manage Lambda functions that can describe cluster resources.

The Lambda function can be invoked to retrieve cluster IAM and VPC
resource information in JSON format.`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newInvokeCommand())
	cmd.AddCommand(newDeleteCommand())

	return cmd
}
