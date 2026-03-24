package clusteriam

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/crypto"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/services/clusteriam"
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

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create service request
	req := &clusteriam.CreateIAMRequest{
		ClusterName:    opts.clusterName,
		OIDCIssuerURL:  opts.oidcIssuerURL,
		OIDCThumbprint: thumbprint,
		AWSConfig:      cfg,
	}

	fmt.Println("📄 Preparing IAM CloudFormation operation...")
	fmt.Printf("☁️  Creating or updating CloudFormation stack: rosa-%s-iam\n", opts.clusterName)
	fmt.Println("   This may take several minutes...")
	fmt.Println()

	// Call service layer
	resp, err := clusteriam.CreateIAM(ctx, req)
	if err != nil {
		return err
	}

	fmt.Println("✅ Cluster IAM resources created successfully!")
	fmt.Printf("   Stack ID: %s\n", resp.StackID)
	fmt.Println()

	if len(resp.Outputs) > 0 {
		fmt.Println("Created Resources:")
		for key, value := range resp.Outputs {
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
