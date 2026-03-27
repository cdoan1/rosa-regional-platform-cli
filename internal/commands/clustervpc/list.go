package clustervpc

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
		Short: "List cluster VPC stacks",
		Long: `List all cluster VPC CloudFormation stacks in the current AWS account.

Example:
  rosactl cluster-vpc list --region us-east-1`,
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

	// List stacks with prefix
	stacks, err := cfnClient.ListStacks(ctx, "rosa-")
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	// Filter VPC stacks
	var vpcStacks []cloudformation.StackInfo
	for _, stack := range stacks {
		if strings.HasSuffix(stack.StackName, "-vpc") {
			vpcStacks = append(vpcStacks, stack)
		}
	}

	if len(vpcStacks) == 0 {
		fmt.Println("No cluster VPC stacks found.")
		return nil
	}

	// Print table
	fmt.Printf("%-30s %-20s %-30s\n", "CLUSTER NAME", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 82))
	for _, stack := range vpcStacks {
		clusterName := strings.TrimPrefix(stack.StackName, "rosa-")
		clusterName = strings.TrimSuffix(clusterName, "-vpc")
		createdTime := ""
		if stack.CreationTime != nil {
			createdTime = stack.CreationTime.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-30s %-20s %-30s\n", clusterName, stack.Status, createdTime)
	}

	return nil
}
