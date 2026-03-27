package cluster

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	pkgconfig "github.com/openshift-online/rosa-regional-platform-cli/internal/config"
	"github.com/spf13/cobra"
)

type createOptions struct {
	clusterName       string
	region            string
	targetProjectID   string
	version           string
	computeReplicas   int
	computeMachineType string
	placementCluster  string
	provider          string
	multiAZ           bool
	labelEnvironment  string
	labelTeam         string
	dryRun            bool
	outputFile        string
	payloadFile       string
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{
		region:             "us-east-1",
		targetProjectID:    "",
		version:            "4.14",
		computeReplicas:    3,
		computeMachineType: "m5.xlarge",
		placementCluster:   "mc01",
		provider:           "aws",
		multiAZ:            true,
		labelEnvironment:   "dev",
		labelTeam:          "platform",
	}

	cmd := &cobra.Command{
		Use:   "create CLUSTER_NAME",
		Short: "Create a cluster configuration or submit cluster to platform API",
		Long: `Create a cluster configuration by gathering IAM and VPC information,
or submit a cluster configuration to the platform API.

Two modes:
1. --dry-run: Generate cluster configuration from CloudFormation stacks
2. --payload: POST a cluster configuration file to the platform API

Examples:
  # Generate cluster configuration (dry-run mode)
  rosactl cluster create my-cluster --region us-east-1 --dry-run
  rosactl cluster create my-cluster --region us-east-1 --dry-run --output-file my-cluster.json

  # Submit cluster to platform API (payload mode)
  rosactl cluster create my-cluster --region us-east-1 --payload my-cluster.json
  rosactl cluster create my-cluster --region us-east-1 --payload my-cluster.json --placement mgmt-cluster-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]

			// Validate mode: must have either --dry-run or --payload
			if opts.dryRun && opts.payloadFile != "" {
				return fmt.Errorf("cannot use both --dry-run and --payload flags")
			}

			if !opts.dryRun && opts.payloadFile == "" {
				return fmt.Errorf("must specify either --dry-run or --payload flag")
			}

			// Set default output file if in dry-run mode and not specified
			if opts.dryRun && opts.outputFile == "" {
				opts.outputFile = fmt.Sprintf("%s-cluster.json", opts.clusterName)
			}

			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", opts.region, "AWS region")
	cmd.Flags().StringVar(&opts.targetProjectID, "target-project-id", opts.targetProjectID, "Target project ID (dry-run mode only)")
	cmd.Flags().StringVar(&opts.version, "version", opts.version, "OpenShift version (dry-run mode only)")
	cmd.Flags().IntVar(&opts.computeReplicas, "compute-replicas", opts.computeReplicas, "Number of compute replicas (dry-run mode only)")
	cmd.Flags().StringVar(&opts.computeMachineType, "compute-machine-type", opts.computeMachineType, "Compute machine type (dry-run mode only)")
	cmd.Flags().StringVar(&opts.placementCluster, "placement", opts.placementCluster, "Management cluster name (overrides payload value)")
	cmd.Flags().StringVar(&opts.provider, "provider", opts.provider, "Cloud provider (dry-run mode only)")
	cmd.Flags().BoolVar(&opts.multiAZ, "multi-az", opts.multiAZ, "Enable multi-AZ deployment (dry-run mode only)")
	cmd.Flags().StringVar(&opts.labelEnvironment, "label-environment", opts.labelEnvironment, "Environment label (dry-run mode only)")
	cmd.Flags().StringVar(&opts.labelTeam, "label-team", opts.labelTeam, "Team label (dry-run mode only)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Generate cluster configuration without submitting to API")
	cmd.Flags().StringVar(&opts.outputFile, "output-file", "", "Output file for cluster configuration (dry-run mode, default: <cluster-name>-cluster.json)")
	cmd.Flags().StringVar(&opts.payloadFile, "payload", "", "JSON payload file to POST to platform API")

	return cmd
}

// toCamelCase converts a PascalCase string to camelCase
// Examples: "VpcId" -> "vpcId", "OIDCProviderArn" -> "oidcProviderArn"
func toCamelCase(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)

	// Find the end of the leading uppercase sequence
	// For "OIDCProviderArn", we want to lowercase "OIDC" but keep "P" uppercase
	uppercaseEnd := 0
	for i := 0; i < len(runes); i++ {
		if !unicode.IsUpper(runes[i]) {
			// Found a non-uppercase character
			break
		}
		uppercaseEnd = i
	}

	// If we have multiple uppercase letters followed by a lowercase letter,
	// we should keep the last uppercase as-is (it starts the next word)
	// e.g., "OIDCProvider" -> uppercase until 'P', keep 'P' uppercase
	if uppercaseEnd > 0 && uppercaseEnd+1 < len(runes) && unicode.IsLower(runes[uppercaseEnd+1]) {
		uppercaseEnd--
	}

	// Convert the leading uppercase sequence to lowercase
	var result strings.Builder
	for i := 0; i <= uppercaseEnd; i++ {
		result.WriteRune(unicode.ToLower(runes[i]))
	}

	// Append the rest unchanged
	for i := uppercaseEnd + 1; i < len(runes); i++ {
		result.WriteRune(runes[i])
	}

	return result.String()
}

func runCreate(ctx context.Context, opts *createOptions) error {
	if opts.payloadFile != "" {
		// Payload mode: POST to platform API
		return runCreateWithPayload(ctx, opts)
	}

	// Dry-run mode: Generate configuration from CloudFormation stacks
	return runCreateDryRun(ctx, opts)
}

func runCreateDryRun(ctx context.Context, opts *createOptions) error {
	// Load AWS config
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Get IAM stack information
	iamStackName := fmt.Sprintf("rosa-%s-iam", opts.clusterName)
	iamStack, err := cfnClient.DescribeStack(ctx, iamStackName)
	if err != nil {
		return fmt.Errorf("failed to describe IAM stack: %w", err)
	}

	// Get VPC stack information
	vpcStackName := fmt.Sprintf("rosa-%s-vpc", opts.clusterName)
	vpcStack, err := cfnClient.DescribeStack(ctx, vpcStackName)
	if err != nil {
		return fmt.Errorf("failed to describe VPC stack: %w", err)
	}

	// Build the spec object with base fields
	spec := map[string]interface{}{
		"provider":             opts.provider,
		"region":               opts.region,
		"version":              opts.version,
		"multi_az":             opts.multiAZ,
		"compute_machine_type": opts.computeMachineType,
		"compute_replicas":     opts.computeReplicas,
		"placement":            opts.placementCluster,
	}

	// Merge IAM outputs directly into spec with camelCase keys
	for key, value := range iamStack.Outputs {
		camelKey := toCamelCase(key)
		spec[camelKey] = value
	}

	// Merge VPC outputs directly into spec with camelCase keys
	for key, value := range vpcStack.Outputs {
		camelKey := toCamelCase(key)
		spec[camelKey] = value
	}

	// Build labels
	labels := map[string]interface{}{
		"environment": opts.labelEnvironment,
		"team":        opts.labelTeam,
		"region":      opts.region,
	}

	// Build the cluster object
	clusterObj := map[string]interface{}{
		"kind":              "Cluster",
		"name":              opts.clusterName,
		"target_project_id": opts.targetProjectID,
		"labels":            labels,
		"spec":              spec,
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(clusterObj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cluster object: %w", err)
	}

	// Print to stdout
	fmt.Println(string(jsonBytes))

	// Save to file
	if err := os.WriteFile(opts.outputFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Print confirmation message to stderr so it doesn't interfere with stdout JSON
	fmt.Fprintf(os.Stderr, "\n✓ Cluster configuration saved to: %s\n", opts.outputFile)

	return nil
}

func runCreateWithPayload(ctx context.Context, opts *createOptions) error {
	// Read the payload file
	payloadBytes, err := os.ReadFile(opts.payloadFile)
	if err != nil {
		return fmt.Errorf("failed to read payload file: %w", err)
	}

	// Validate JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("invalid JSON in payload file: %w", err)
	}

	// Override placement if specified (and not the default)
	// We check against default to avoid unnecessary override messages
	if opts.placementCluster != "" {
		// Access the spec object and update placement
		if spec, ok := payload["spec"].(map[string]interface{}); ok {
			currentPlacement := spec["placement"]
			if currentPlacement != opts.placementCluster {
				spec["placement"] = opts.placementCluster
				fmt.Fprintf(os.Stderr, "Overriding placement: %v → %s\n", currentPlacement, opts.placementCluster)
			}
		}
	}

	// Re-marshal the payload with any overrides
	payloadBytes, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal modified payload: %w", err)
	}

	// Get the platform API URL from config
	baseURL, err := pkgconfig.GetPlatformAPIURL()
	if err != nil {
		return err
	}

	// Build the API endpoint URL
	endpoint := fmt.Sprintf("%s/api/v0/clusters", baseURL)

	// Load AWS config for SigV4 signing
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Calculate SHA256 hash of the request body
	hash := sha256.Sum256(payloadBytes)
	payloadHash := hex.EncodeToString(hash[:])

	// Sign the request with AWS SigV4
	signer := v4.NewSigner()
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Determine the region from AWS config
	region := cfg.Region
	if region == "" {
		region = opts.region
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	err = signer.SignHTTP(ctx, creds, req, payloadHash, "execute-api", region, time.Now())
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute the request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Pretty print the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If we can't parse as JSON, just print the raw response
		fmt.Println(string(body))
		return nil
	}

	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// Fall back to raw response
		fmt.Println(string(body))
		return nil
	}

	fmt.Println(string(prettyJSON))
	fmt.Fprintf(os.Stderr, "\n✓ Cluster created successfully\n")

	return nil
}
