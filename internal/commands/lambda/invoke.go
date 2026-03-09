package lambda

import (
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

var invokeVersion string

func newInvokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke <name>",
		Short: "Invoke a Lambda function",
		Long:  "Execute an AWS Lambda function and display its output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			functionName := args[0]

			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			result, logs, err := client.InvokeFunction(cmd.Context(), functionName, invokeVersion)
			if err != nil {
				return err
			}

			fmt.Println(result)

			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose && logs != "" {
				fmt.Println("\nExecution logs:")
				fmt.Println(logs)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&invokeVersion, "version", "", "Invoke specific version (default: $LATEST)")

	return cmd
}
