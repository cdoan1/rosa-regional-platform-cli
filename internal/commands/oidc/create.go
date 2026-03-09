package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var region string
	var functionName string

	cmd := &cobra.Command{
		Use:   "create <bucket-name>",
		Short: "Create a new S3-backed OIDC issuer",
		Long: `Create a new S3-backed OIDC issuer with an IAM OIDC provider.

This command invokes the OIDC Lambda function to:
1. Create an S3 bucket to host OIDC discovery documents
2. Upload .well-known/openid-configuration and keys.json
3. Create an IAM OIDC provider pointing to the S3 bucket

The RSA key pair is embedded in the Lambda function during creation.

Bucket naming: The provided name will be automatically prefixed with "oidc-issuer-"
(e.g., "my-cluster" becomes "oidc-issuer-my-cluster"). The final bucket name
must be globally unique across AWS.`,
		Example: `  # Create OIDC issuer with auto-detected region (auto-prefixed to oidc-issuer-my-cluster)
  rosactl oidc create my-cluster

  # Create OIDC issuer in specific region
  rosactl oidc create my-cluster --region us-west-2

  # Use explicit bucket name (will still be prefixed)
  rosactl oidc create hcp-prod --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bucketName := args[0]

			// Auto-prefix bucket name with "oidc-issuer-" if not already present
			if !strings.HasPrefix(bucketName, "oidc-issuer-") {
				bucketName = "oidc-issuer-" + bucketName
				fmt.Printf("Using bucket name: %s\n", bucketName)
			}

			ctx := context.Background()

			// Create Lambda client
			lambdaClient, err := lambda.NewClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create Lambda client: %w", err)
			}

			// Determine region
			if region == "" {
				region = os.Getenv("AWS_REGION")
				if region == "" {
					region = "us-east-1"
				}
			}

			// Construct payload (JWK is already in Lambda environment)
			payload := map[string]interface{}{
				"bucket_name": bucketName,
				"region":      region,
			}

			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}

			// Invoke OIDC Lambda function
			fmt.Printf("Invoking OIDC Lambda function '%s'...\n", functionName)
			result, err := lambdaClient.InvokeFunctionWithPayload(ctx, functionName, payloadBytes)
			if err != nil {
				return fmt.Errorf("failed to invoke Lambda function: %w", err)
			}

			// Parse response
			var response struct {
				StatusCode int    `json:"statusCode"`
				Body       string `json:"body"`
			}

			if err := json.Unmarshal(result, &response); err != nil {
				// If unmarshaling fails, just print raw response
				fmt.Println("Lambda response:")
				fmt.Println(string(result))
				return nil
			}

			// Parse body
			var body map[string]interface{}
			if err := json.Unmarshal([]byte(response.Body), &body); err == nil {
				// Pretty print the body
				prettyBody, _ := json.MarshalIndent(body, "", "  ")
				fmt.Println("\nOIDC Issuer created successfully:")
				fmt.Println(string(prettyBody))

				// Check for error in response
				if response.StatusCode != 200 {
					if errMsg, ok := body["error"].(string); ok {
						return fmt.Errorf("lambda returned error: %s", errMsg)
					}
				}
			} else {
				// Fallback to raw body
				fmt.Println("\nOIDC Issuer created:")
				fmt.Println(response.Body)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&region, "region", "", "AWS region for the S3 bucket (defaults to AWS_REGION or us-east-1)")
	cmd.Flags().StringVar(&functionName, "function", "oidc", "Name of the OIDC Lambda function to invoke")

	return cmd
}
