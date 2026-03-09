package lambda

import (
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a Lambda function",
		Long:  "Update an existing AWS Lambda function with datetime handler that shows current time, creation time, and ago delta.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			functionName := args[0]

			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			version, err := client.UpdateFunctionCode(cmd.Context(), functionName)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully updated Lambda function: %s\n", functionName)
			fmt.Printf("Published version: %s\n", version)
			fmt.Println("Function now returns current time, creation time, and ago delta")
			return nil
		},
	}

	return cmd
}
