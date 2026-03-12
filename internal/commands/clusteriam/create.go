package clusteriam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/cloudformation/templates"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/crypto"
	"github.com/spf13/cobra"
)

type createOptions struct {
	clusterName   string
	oidcIssuerURL string
	region        string
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create CLUSTER_NAME",
		Short: "Create cluster IAM resources",
		Long: `Create IAM OIDC provider and roles for a hosted cluster.

This command:
1. Fetches the TLS thumbprint from the OIDC issuer URL
2. Creates a CloudFormation stack with the following resources:
   - IAM OIDC Provider
   - 7 control plane IAM roles (ingress, cloud-controller-manager, ebs-csi, etc.)
   - Worker node IAM role and instance profile

Example:
  rosactl cluster-iam create my-cluster \
    --oidc-issuer-url https://d1234.cloudfront.net/my-cluster \
    --region us-east-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.oidcIssuerURL, "oidc-issuer-url", "", "OIDC issuer URL from Management Cluster (required)")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")

	cmd.MarkFlagRequired("oidc-issuer-url")
	cmd.MarkFlagRequired("region")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	// Validate cluster name
	if err := validateClusterName(opts.clusterName); err != nil {
		return err
	}

	// Validate OIDC issuer URL
	if !strings.HasPrefix(opts.oidcIssuerURL, "https://") {
		return fmt.Errorf("OIDC issuer URL must start with https://")
	}

	fmt.Println("🔐 Creating cluster IAM resources...")
	fmt.Printf("   Cluster: %s\n", opts.clusterName)
	fmt.Printf("   OIDC Issuer: %s\n", opts.oidcIssuerURL)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Println()

	// Fetch TLS thumbprint
	fmt.Println("🔍 Fetching TLS thumbprint from OIDC issuer...")
	thumbprint, err := crypto.GetOIDCThumbprint(ctx, opts.oidcIssuerURL)
	if err != nil {
		return fmt.Errorf("failed to fetch TLS thumbprint: %w", err)
	}
	fmt.Printf("   Thumbprint: %s\n", thumbprint)
	fmt.Println()

	// Derive OIDC issuer domain (remove https:// prefix)
	oidcIssuerDomain, err := crypto.GetOIDCIssuerDomain(opts.oidcIssuerURL)
	if err != nil {
		return fmt.Errorf("failed to parse OIDC issuer URL: %w", err)
	}

	// Read CloudFormation template
	fmt.Println("📄 Loading CloudFormation template...")
	templateBody, err := templates.Read("cluster-iam.yaml")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Prepare stack parameters
	stackName := fmt.Sprintf("rosa-%s-iam", opts.clusterName)
	params := &cloudformation.CreateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":      opts.clusterName,
			"OIDCIssuerURL":    opts.oidcIssuerURL,
			"OIDCIssuerDomain": oidcIssuerDomain,
			"OIDCThumbprint":   thumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("Cluster"),
				Value: aws.String(opts.clusterName),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("rosactl"),
			},
			{
				Key:   aws.String("red-hat-managed"),
				Value: aws.String("true"),
			},
		},
		WaitTimeout: 15 * time.Minute,
	}

	fmt.Printf("☁️  Creating CloudFormation stack: %s\n", stackName)
	fmt.Println("   This may take several minutes...")
	fmt.Println()

	// Create stack
	output, err := cfnClient.CreateStack(ctx, params)
	if err != nil {
		// Check if stack already exists, try update instead
		var alreadyExistsErr *cloudformation.StackAlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			fmt.Println("ℹ️  Stack already exists, attempting update...")
			return updateStack(ctx, cfnClient, opts, stackName, templateBody, oidcIssuerDomain, thumbprint)
		}

		// Get stack events to show what failed
		fmt.Println()
		fmt.Println("❌ Stack creation failed. Recent events:")
		events, evtErr := cfnClient.GetStackEvents(ctx, stackName, 10)
		if evtErr == nil && len(events) > 0 {
			for _, event := range events {
				if event.ResourceStatus == "CREATE_FAILED" || event.ResourceStatusReason != "" {
					fmt.Printf("   • %s: %s", event.LogicalResourceID, event.ResourceStatus)
					if event.ResourceStatusReason != "" {
						fmt.Printf(" - %s", event.ResourceStatusReason)
					}
					fmt.Println()
				}
			}
		}
		fmt.Println()

		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Println("✅ Cluster IAM resources created successfully!")
	fmt.Printf("   Stack ID: %s\n", output.StackID)
	fmt.Println()

	if len(output.Outputs) > 0 {
		fmt.Println("Created Resources:")
		for key, value := range output.Outputs {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

// updateStack updates an existing cluster IAM CloudFormation stack
func updateStack(ctx context.Context, cfnClient *cloudformation.Client, opts *createOptions,
	stackName, templateBody, oidcIssuerDomain, thumbprint string) error {
	updateParams := &cloudformation.UpdateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":      opts.clusterName,
			"OIDCIssuerURL":    opts.oidcIssuerURL,
			"OIDCIssuerDomain": oidcIssuerDomain,
			"OIDCThumbprint":   thumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		WaitTimeout: 15 * time.Minute,
	}

	fmt.Println("   Updating stack...")
	output, err := cfnClient.UpdateStack(ctx, updateParams)
	if err != nil {
		// Check if no changes are needed
		var noChangesErr *cloudformation.NoChangesError
		if errors.As(err, &noChangesErr) {
			fmt.Println("ℹ️  No changes needed, stack is up to date")
			// Still get outputs
			stackOutput, err := cfnClient.GetStackOutputs(ctx, stackName)
			if err != nil {
				return fmt.Errorf("failed to get stack outputs: %w", err)
			}

			fmt.Println()
			fmt.Println("Stack Resources:")
			for key, value := range stackOutput.Outputs {
				fmt.Printf("  %s: %s\n", key, value)
			}
			return nil
		}
		return fmt.Errorf("failed to update stack: %w", err)
	}

	fmt.Println("✅ Cluster IAM resources updated successfully!")
	fmt.Printf("   Stack ID: %s\n", output.StackID)
	fmt.Println()

	if len(output.Outputs) > 0 {
		fmt.Println("Updated Resources:")
		for key, value := range output.Outputs {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

func validateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	// Cluster name must be lowercase alphanumeric with hyphens
	for i, c := range name {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' && i > 0 {
			continue
		}
		if c == '-' && i > 0 && i < len(name)-1 {
			continue
		}
		return fmt.Errorf("cluster name must be lowercase alphanumeric with hyphens, got: %s", name)
	}

	return nil
}
