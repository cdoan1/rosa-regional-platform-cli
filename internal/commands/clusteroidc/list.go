package clusteroidc

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
		Short: "List all cluster OIDC provider stacks",
		Long: `List all CloudFormation stacks for cluster OIDC providers.

Example:
  rosactl oidc list --region us-east-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	cfnClient := cloudformation.NewClient(cfg)

	stacks, err := cfnClient.ListStacks(ctx, "rosa-")
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	var oidcStacks []cloudformation.StackInfo
	for _, stack := range stacks {
		if strings.HasSuffix(stack.StackName, "-oidc") {
			oidcStacks = append(oidcStacks, stack)
		}
	}

	if len(oidcStacks) == 0 {
		fmt.Println("No cluster OIDC provider stacks found")
		return nil
	}

	fmt.Printf("%-30s %-20s %-20s\n", "CLUSTER NAME", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 70))

	for _, stack := range oidcStacks {
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
	// Stack name format: rosa-<cluster-name>-oidc
	if strings.HasPrefix(stackName, "rosa-") && strings.HasSuffix(stackName, "-oidc") {
		return stackName[5 : len(stackName)-5]
	}
	return stackName
}
