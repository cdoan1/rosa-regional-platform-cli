package clusteriam

import (
	"context"
	"fmt"

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
		Short: "Describe cluster IAM resources",
		Long: `Show details about cluster IAM resources.

This command displays the CloudFormation stack status and all outputs
(role ARNs, OIDC provider ARN, instance profile name).

Example:
  rosactl cluster-iam describe my-cluster --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runDescribe(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")

	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func runDescribe(ctx context.Context, opts *describeOptions) error {
	stackName := fmt.Sprintf("rosa-%s-iam", opts.clusterName)

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Get stack information
	stack, err := cfnClient.DescribeStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("failed to describe stack: %w", err)
	}

	// Display stack information
	fmt.Printf("📋 Stack: %s\n", stack.StackName)
	fmt.Printf("   Cluster: %s\n", opts.clusterName)
	fmt.Printf("   Status: %s\n", stack.Status)
	if stack.CreationTime != nil {
		fmt.Printf("   Created: %s\n", stack.CreationTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	if len(stack.Outputs) > 0 {
		fmt.Println("IAM Resources:")

		// Group outputs by type
		var oidcProvider, controlPlaneRoles, workerResources []string

		for key, value := range stack.Outputs {
			switch key {
			case "OIDCProviderArn", "OIDCProviderURL":
				oidcProvider = append(oidcProvider, fmt.Sprintf("  %s: %s", key, value))
			case "WorkerRoleArn", "WorkerInstanceProfileName":
				workerResources = append(workerResources, fmt.Sprintf("  %s: %s", key, value))
			default:
				controlPlaneRoles = append(controlPlaneRoles, fmt.Sprintf("  %s: %s", key, value))
			}
		}

		if len(oidcProvider) > 0 {
			fmt.Println("\n  OIDC Provider:")
			for _, line := range oidcProvider {
				fmt.Println(line)
			}
		}

		if len(controlPlaneRoles) > 0 {
			fmt.Println("\n  Control Plane Roles:")
			for _, line := range controlPlaneRoles {
				fmt.Println(line)
			}
		}

		if len(workerResources) > 0 {
			fmt.Println("\n  Worker Resources:")
			for _, line := range workerResources {
				fmt.Println(line)
			}
		}
	}

	return nil
}
