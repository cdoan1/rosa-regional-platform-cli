package lambda

import (
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a Lambda function",
		Long:  "Remove an AWS Lambda function.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			functionName := args[0]

			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			if err := client.DeleteFunction(cmd.Context(), functionName); err != nil {
				return err
			}

			fmt.Printf("Successfully deleted Lambda function: %s\n", functionName)
			return nil
		},
	}
}
