package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/config"
	"github.com/spf13/cobra"
)

type listOptions struct {
	limit  int
	offset int
	status string
}

func newListCommand() *cobra.Command {
	opts := &listOptions{
		limit:  50,
		offset: 0,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List clusters from the platform API",
		Long: `List clusters from the platform API.

This command queries the platform API to retrieve a list of clusters.

Example:
  rosactl cluster list
  rosactl cluster list --limit 10
  rosactl cluster list --status Ready`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum number of clusters to return (1-100)")
	cmd.Flags().IntVar(&opts.offset, "offset", opts.offset, "Number of clusters to skip")
	cmd.Flags().StringVar(&opts.status, "status", opts.status, "Filter by status (Pending, Progressing, Ready, Failed)")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	// Get the platform API URL from config
	baseURL, err := config.GetPlatformAPIURL()
	if err != nil {
		return err
	}

	// Build the API endpoint URL
	endpoint := fmt.Sprintf("%s/api/v0/clusters?limit=%d&offset=%d", baseURL, opts.limit, opts.offset)
	if opts.status != "" {
		endpoint = fmt.Sprintf("%s&status=%s", endpoint, opts.status)
	}

	// Load AWS config for SigV4 signing
	cfg, err := aws.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Sign the request with AWS SigV4
	signer := v4.NewSigner()
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Determine the region from AWS config
	region := cfg.Region
	if region == "" {
		region = "us-east-1" // Default region
	}

	// SHA256 hash of empty body for GET requests
	emptyBodyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	err = signer.SignHTTP(ctx, creds, req, emptyBodyHash, "execute-api", region, time.Now())
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
	defer func() { _ = resp.Body.Close() }()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error responses
	if resp.StatusCode != http.StatusOK {
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
	return nil
}
