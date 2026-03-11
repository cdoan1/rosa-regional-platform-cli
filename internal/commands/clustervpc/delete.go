package clustervpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	clusterName    string
	region         string
	lambdaFunction string
}

func newDeleteCommand() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete CLUSTER_NAME",
		Short: "Delete cluster VPC resources",
		Long: `Delete VPC networking resources for a hosted cluster.

This command invokes the Lambda function to delete the CloudFormation stack
containing all VPC resources.

Example:
  rosactl cluster-vpc delete my-cluster --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.Flags().StringVar(&opts.lambdaFunction, "lambda-function", defaultLambdaFunction, "Name of the Lambda function")

	cmd.MarkFlagRequired("region")

	return cmd
}

func runDelete(ctx context.Context, opts *deleteOptions) error {
	fmt.Printf("Deleting cluster VPC resources for: %s\n", opts.clusterName)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Println()

	// Create Lambda client
	lambdaClient, err := lambda.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Prepare Lambda payload
	payload := map[string]string{
		"action":       "delete-cluster-vpc",
		"cluster_name": opts.clusterName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Printf("Invoking Lambda function: %s\n", opts.lambdaFunction)

	// Invoke Lambda
	result, err := lambdaClient.InvokeFunctionWithPayload(ctx, opts.lambdaFunction, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	// Parse response
	var response lambdaResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse Lambda response: %w", err)
	}

	// Check for errors
	if response.Error != "" {
		return fmt.Errorf("Lambda execution failed: %s", response.Error)
	}

	fmt.Println("Cluster VPC resources deleted successfully!")

	return nil
}
