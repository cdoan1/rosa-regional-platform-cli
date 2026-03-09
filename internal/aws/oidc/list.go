package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type ProviderInfo struct {
	ARN        string   `json:"arn"`
	URL        string   `json:"url"`
	ClientIDs  []string `json:"client_ids"`
	Thumbprint []string `json:"thumbprint"`
	CreateDate string   `json:"create_date"`
}

// ListProviders lists all IAM OIDC providers
func (c *Client) ListProviders(ctx context.Context) ([]ProviderInfo, error) {
	listOutput, err := c.iam.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, &OIDCError{
			Operation: "list",
			Message:   fmt.Sprintf("failed to list OIDC providers: %v", err),
		}
	}

	var providers []ProviderInfo

	for _, provider := range listOutput.OpenIDConnectProviderList {
		providerARN := aws.ToString(provider.Arn)

		getOutput, err := c.iam.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: aws.String(providerARN),
		})
		if err != nil {
			continue // Skip providers we can't read
		}

		url := extractURLFromARN(providerARN)

		providers = append(providers, ProviderInfo{
			ARN:        providerARN,
			URL:        url,
			ClientIDs:  getOutput.ClientIDList,
			Thumbprint: getOutput.ThumbprintList,
			CreateDate: aws.ToTime(getOutput.CreateDate).Format("2006-01-02 15:04:05"),
		})
	}

	return providers, nil
}

// extractURLFromARN extracts the issuer URL from an OIDC provider ARN
// ARN format: arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com
func extractURLFromARN(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/")
	}
	return ""
}

// DeleteProvider deletes an IAM OIDC provider by ARN
func (c *Client) DeleteProvider(ctx context.Context, arn string) error {
	_, err := c.iam.DeleteOpenIDConnectProvider(ctx, &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(arn),
	})
	if err != nil {
		return &OIDCError{
			Operation: "delete",
			Message:   fmt.Sprintf("failed to delete OIDC provider: %v", err),
		}
	}
	return nil
}
