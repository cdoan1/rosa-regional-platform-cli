package bootstrap

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	region    string
	stackName string
}

func newStatusCommand() *cobra.Command {
	opts := &statusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of the Lambda function bootstrap stack",
		Long: `Check the status of the Lambda function infrastructure.

This command shows the current state of the CloudFormation stack and its outputs.

Example:
  rosactl bootstrap status --region us-east-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.Flags().StringVar(&opts.stackName, "stack-name", defaultStackName, "Name of the CloudFormation stack")

	cmd.MarkFlagRequired("region")

	return cmd
}

func runStatus(ctx context.Context, opts *statusOptions) error {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Get stack information
	stack, err := cfnClient.DescribeStack(ctx, opts.stackName)
	if err != nil {
		return fmt.Errorf("failed to describe stack: %w", err)
	}

	// Display stack status
	fmt.Printf("📋 Stack: %s\n", stack.StackName)
	fmt.Printf("   Status: %s\n", stack.Status)
	if stack.CreationTime != nil {
		fmt.Printf("   Created: %s\n", stack.CreationTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	if len(stack.Outputs) > 0 {
		fmt.Println("Outputs:")
		for key, value := range stack.Outputs {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}
