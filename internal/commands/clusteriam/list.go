package clusteriam

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/spf13/cobra"
)

type listOptions struct {
	region string
}

func newListCommand() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all cluster IAM stacks",
		Long: `List all CloudFormation stacks for cluster IAM resources.

Example:
  rosactl cluster-iam list --region us-east-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")

	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// List stacks with "rosa-" prefix
	stacks, err := cfnClient.ListStacks(ctx, "rosa-")
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	// Filter IAM stacks
	var iamStacks []cloudformation.StackInfo
	for _, stack := range stacks {
		if strings.HasSuffix(stack.StackName, "-iam") {
			iamStacks = append(iamStacks, stack)
		}
	}

	if len(iamStacks) == 0 {
		fmt.Println("No cluster IAM stacks found")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %-20s %-20s\n", "CLUSTER NAME", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 70))

	// Print stacks
	for _, stack := range iamStacks {
		// Extract cluster name from stack name (format: rosa-<cluster-name>-iam)
		clusterName := extractClusterName(stack.StackName)

		createdTime := ""
		if stack.CreationTime != nil {
			createdTime = stack.CreationTime.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("%-30s %-20s %-20s\n", clusterName, stack.Status, createdTime)
	}

	return nil
}

func extractClusterName(stackName string) string {
	// Stack name format: rosa-<cluster-name>-iam
	if strings.HasPrefix(stackName, "rosa-") && strings.HasSuffix(stackName, "-iam") {
		return stackName[5 : len(stackName)-4]
	}
	return stackName
}
