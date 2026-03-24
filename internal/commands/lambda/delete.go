package lambda

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	functionName string
	region       string
	stackName    string
}

func newDeleteCommand() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete FUNCTION_NAME",
		Short: "Delete a Lambda function",
		Long: `Delete a Lambda function and its CloudFormation stack.

This command removes the Lambda function and all associated resources
created by the CloudFormation stack.

Example:
  rosactl lambda delete my-cluster-lambda --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.functionName = args[0]
			opts.stackName = fmt.Sprintf("rosa-lambda-%s", opts.functionName)
			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.MarkFlagRequired("region")

	return cmd
}

func runDelete(ctx context.Context, opts *deleteOptions) error {
	fmt.Printf("Deleting Lambda function: %s\n", opts.functionName)
	fmt.Printf("   Stack name: %s\n", opts.stackName)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Println()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Delete stack
	fmt.Println("Deleting CloudFormation stack...")
	err = cfnClient.DeleteStack(ctx, opts.stackName, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to delete stack: %w", err)
	}

	fmt.Println()
	fmt.Println("Lambda function deleted successfully!")

	return nil
}
