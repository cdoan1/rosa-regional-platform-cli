package clusteriam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	clusterName string
	region      string
}

func newDeleteCommand() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete CLUSTER_NAME",
		Short: "Delete cluster IAM resources",
		Long: `Delete IAM OIDC provider and roles for a hosted cluster.

This command deletes the CloudFormation stack containing all cluster IAM resources.

Example:
  rosactl cluster-iam delete my-cluster --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")

	cmd.MarkFlagRequired("region")

	return cmd
}

func runDelete(ctx context.Context, opts *deleteOptions) error {
	fmt.Printf("🗑️  Deleting cluster IAM resources for: %s\n", opts.clusterName)
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
	stackName := fmt.Sprintf("rosa-%s-iam", opts.clusterName)

	fmt.Printf("☁️  Deleting CloudFormation stack: %s\n", stackName)
	fmt.Println("   This may take several minutes...")
	fmt.Println()

	err = cfnClient.DeleteStack(ctx, stackName, 15*time.Minute)
	if err != nil {
		// Check if stack doesn't exist
		var notFoundErr *cloudformation.StackNotFoundError
		if errors.As(err, &notFoundErr) {
			fmt.Println("ℹ️  Stack not found, may have been already deleted")
			return nil
		}
		return fmt.Errorf("failed to delete stack: %w", err)
	}

	fmt.Println("✅ Cluster IAM resources deleted successfully!")

	return nil
}
