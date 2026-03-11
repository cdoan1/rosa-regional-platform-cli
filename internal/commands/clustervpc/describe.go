package clustervpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/spf13/cobra"
)

type describeOptions struct {
	clusterName string
	region      string
}

func newDescribeCommand() *cobra.Command {
	opts := &describeOptions{}

	cmd := &cobra.Command{
		Use:   "describe CLUSTER_NAME",
		Short: "Describe cluster VPC stack",
		Long: `Show detailed information about a cluster VPC CloudFormation stack.

Example:
  rosactl cluster-vpc describe my-cluster --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runDescribe(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.MarkFlagRequired("region")

	return cmd
}

func runDescribe(ctx context.Context, opts *describeOptions) error {
	stackName := fmt.Sprintf("rosa-%s-vpc", opts.clusterName)

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Describe stack
	stack, err := cfnClient.DescribeStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("failed to describe stack: %w", err)
	}

	// Print stack details
	fmt.Printf("Cluster VPC Stack: %s\n", opts.clusterName)
	fmt.Println()
	fmt.Printf("Stack Name: %s\n", stack.StackName)
	fmt.Printf("Stack ID: %s\n", stack.StackID)
	fmt.Printf("Status: %s\n", stack.Status)
	if stack.CreationTime != nil {
		fmt.Printf("Created: %s\n", stack.CreationTime.Format("2006-01-02 15:04:05"))
	}

	// Print outputs
	if len(stack.Outputs) > 0 {
		fmt.Println()
		fmt.Println("VPC Resources:")
		fmt.Println(strings.Repeat("-", 80))
		for key, value := range stack.Outputs {
			fmt.Printf("  %-30s: %s\n", key, value)
		}
	}

	return nil
}
