package clusteriam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/crypto"
	"github.com/spf13/cobra"
)

const (
	defaultLambdaFunction = "rosa-regional-platform-lambda"
)

type createOptions struct {
	clusterName    string
	oidcIssuerURL  string
	region         string
	lambdaFunction string
}

type lambdaPayload struct {
	Action          string `json:"action"`
	ClusterName     string `json:"cluster_name"`
	OIDCIssuerURL   string `json:"oidc_issuer_url"`
	OIDCThumbprint  string `json:"oidc_thumbprint"`
}

type lambdaResponse struct {
	StackID  string            `json:"stack_id"`
	Outputs  map[string]string `json:"outputs"`
	Error    string            `json:"error,omitempty"`
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create CLUSTER_NAME",
		Short: "Create cluster IAM resources",
		Long: `Create IAM OIDC provider and roles for a hosted cluster.

This command:
1. Fetches the TLS thumbprint from the OIDC issuer URL
2. Invokes the Lambda function to apply the CloudFormation template
3. Creates the following resources via CloudFormation:
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
	cmd.Flags().StringVar(&opts.lambdaFunction, "lambda-function", defaultLambdaFunction, "Name of the Lambda function")

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

	// Create Lambda client
	lambdaClient, err := lambda.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Prepare Lambda payload
	payload := lambdaPayload{
		Action:         "apply-cluster-iam",
		ClusterName:    opts.clusterName,
		OIDCIssuerURL:  opts.oidcIssuerURL,
		OIDCThumbprint: thumbprint,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Printf("🚀 Invoking Lambda function: %s\n", opts.lambdaFunction)

	// Invoke Lambda
	result, err := lambdaClient.InvokeFunctionWithPayload(ctx, opts.lambdaFunction, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	// Parse response
	var response lambdaResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse Lambda response: %w", err)
	}

	// Check for errors
	if response.Error != "" {
		return fmt.Errorf("Lambda execution failed: %s", response.Error)
	}

	fmt.Println("✅ Cluster IAM resources created successfully!")
	fmt.Println()
	fmt.Println("Created Resources:")
	if len(response.Outputs) > 0 {
		for key, value := range response.Outputs {
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
