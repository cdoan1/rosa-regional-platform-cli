package lambda

import (
	"github.com/spf13/cobra"
)

func NewLambdaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lambda",
		Short: "Manage AWS Lambda functions",
		Long:  "Create, invoke, delete, and list AWS Lambda functions.",
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newInvokeCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newVersionsCommand())

	return cmd
}
