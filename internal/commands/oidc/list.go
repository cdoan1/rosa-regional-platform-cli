package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/oidc"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/s3"
	"github.com/spf13/cobra"
)

type IssuerInfo struct {
	BucketName  string `json:"bucket_name"`
	IssuerURL   string `json:"issuer_url"`
	ProviderARN string `json:"provider_arn,omitempty"`
	Status      string `json:"status"`
	Region      string `json:"region"`
}

func newListCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List OIDC issuers",
		Long:  "Display all S3-backed OIDC issuers and their associated IAM OIDC providers.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			s3Client, err := s3.NewClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create S3 client: %w", err)
			}

			oidcClient, err := oidc.NewClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create OIDC client: %w", err)
			}

			// List S3 buckets with "oidc-issuer-" prefix
			buckets, err := s3Client.ListBucketsWithPrefix(ctx, "oidc-issuer-")
			if err != nil {
				return fmt.Errorf("failed to list S3 buckets: %w", err)
			}

			// List IAM OIDC providers
			providers, err := oidcClient.ListProviders(ctx)
			if err != nil {
				return fmt.Errorf("failed to list OIDC providers: %w", err)
			}

			// Correlate buckets with providers
			issuers := correlateIssuers(buckets, providers)

			if outputFormat == "json" {
				data, err := json.MarshalIndent(issuers, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			// Table format
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)

			if _, err := fmt.Fprintln(w, "BUCKET NAME\tISSUER URL\tSTATUS\tPROVIDER ARN"); err != nil {
				return err
			}
			for _, issuer := range issuers {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					issuer.BucketName,
					issuer.IssuerURL,
					issuer.Status,
					truncateARN(issuer.ProviderARN),
				); err != nil {
					return err
				}
			}

			return w.Flush()
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")

	return cmd
}

func correlateIssuers(buckets []s3.BucketInfo, providers []oidc.ProviderInfo) []IssuerInfo {
	var issuers []IssuerInfo

	// Create a map of provider URLs for quick lookup
	providerMap := make(map[string]oidc.ProviderInfo)
	for _, provider := range providers {
		providerMap[provider.URL] = provider
	}

	// Process each bucket
	for _, bucket := range buckets {
		issuerURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket.Name, bucket.Region)

		issuer := IssuerInfo{
			BucketName: bucket.Name,
			IssuerURL:  issuerURL,
			Region:     bucket.Region,
		}

		// Check if there's a matching OIDC provider
		// The provider URL might be stored without https:// prefix
		providerURL := strings.TrimPrefix(issuerURL, "https://")
		if provider, ok := providerMap[providerURL]; ok {
			issuer.ProviderARN = provider.ARN
			issuer.Status = "Active"
		} else if provider, ok := providerMap[issuerURL]; ok {
			issuer.ProviderARN = provider.ARN
			issuer.Status = "Active"
		} else {
			issuer.Status = "S3 Only"
		}

		issuers = append(issuers, issuer)
	}

	// Find providers without matching S3 buckets
	bucketMap := make(map[string]bool)
	for _, bucket := range buckets {
		issuerURL1 := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket.Name, bucket.Region)
		issuerURL2 := strings.TrimPrefix(issuerURL1, "https://")
		bucketMap[issuerURL1] = true
		bucketMap[issuerURL2] = true
	}

	for _, provider := range providers {
		if !bucketMap[provider.URL] && !bucketMap["https://"+provider.URL] {
			issuers = append(issuers, IssuerInfo{
				BucketName:  extractBucketFromURL(provider.URL),
				IssuerURL:   provider.URL,
				ProviderARN: provider.ARN,
				Status:      "IAM Only",
				Region:      "unknown",
			})
		}
	}

	return issuers
}

func extractBucketFromURL(url string) string {
	// URL format: bucket-name.s3.region.amazonaws.com
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.Split(url, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

func truncateARN(arn string) string {
	if len(arn) == 0 {
		return "-"
	}
	if len(arn) > 60 {
		return arn[:60] + "..."
	}
	return arn
}
