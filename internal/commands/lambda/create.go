package lambda

import (
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var handler string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new Lambda function",
		Long: `Create a new AWS Lambda function.

Handler options (what the function does):
  default     - Basic Python handler that returns 'hello world' (default)
  oidc        - OIDC issuer management (creates S3-backed OIDC providers)
  oidc-delete - OIDC deletion (removes S3 buckets and IAM OIDC providers)`,
		Example: `  # Create a Lambda with default handler
  rosactl lambda create my-function

  # Create an OIDC issuer management function
  rosactl lambda create my-oidc-issuer --handler oidc

  # Create an OIDC deletion function
  rosactl lambda create my-oidc-cleanup --handler oidc-delete`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			functionName := args[0]

			// Validate handler
			validHandlers := map[string]bool{
				"default":     true,
				"oidc":        true,
				"oidc-delete": true,
			}
			if !validHandlers[handler] {
				return fmt.Errorf("invalid handler: %s (valid handlers: default, oidc, oidc-delete)", handler)
			}

			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			var arn string

			// Create function based on handler
			switch handler {
			case "oidc-delete":
				fmt.Println("Creating OIDC deletion Lambda function...")
				arn, err = client.CreateOIDCDeleteFunction(cmd.Context(), functionName)
			case "oidc":
				fmt.Println("Creating OIDC issuer Lambda function...")
				arn, err = client.CreateOIDCFunction(cmd.Context(), functionName)
			default:
				fmt.Println("Creating ZIP-based Lambda function...")
				arn, err = client.CreateFunction(cmd.Context(), functionName)
			}

			if err != nil {
				return err
			}

			fmt.Printf("Successfully created Lambda function: %s\n", functionName)
			fmt.Printf("ARN: %s\n", arn)

			// Show usage instructions for OIDC functions
			switch handler {
			case "oidc-delete":
				fmt.Println("\nTo delete an OIDC issuer, use:")
				fmt.Printf("  rosactl oidc delete <bucket-name> --function %s\n", functionName)
				fmt.Println("\nExample:")
				fmt.Printf("  rosactl oidc delete my-cluster --function %s\n", functionName)
			case "oidc":
				fmt.Println("\nTo create an OIDC issuer, use:")
				fmt.Printf("  rosactl oidc create <bucket-name> --function %s\n", functionName)
				fmt.Println("\nExample:")
				fmt.Printf("  rosactl oidc create my-cluster --function %s\n", functionName)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&handler, "handler", "default", "Function handler: default, oidc, or oidc-delete")

	return cmd
}
