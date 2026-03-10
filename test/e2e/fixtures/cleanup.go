package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// ResourceTracker tracks AWS resources created during tests for cleanup
type ResourceTracker struct {
	lambdaFunctions []string
	oidcProviders   []string
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		lambdaFunctions: make([]string, 0),
		oidcProviders:   make([]string, 0),
	}
}

// TrackLambda adds a Lambda function to the cleanup list
func (t *ResourceTracker) TrackLambda(functionName string) {
	t.lambdaFunctions = append(t.lambdaFunctions, functionName)
}

// TrackOIDCProvider adds an OIDC provider ARN to the cleanup list
func (t *ResourceTracker) TrackOIDCProvider(providerARN string) {
	t.oidcProviders = append(t.oidcProviders, providerARN)
}

// CleanupAll removes all tracked resources
func (t *ResourceTracker) CleanupAll(ctx context.Context, awsHelper *AWSTestHelper) error {
	var errors []error

	// Cleanup Lambda functions
	for _, functionName := range t.lambdaFunctions {
		if err := t.cleanupLambdaFunction(ctx, awsHelper, functionName); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup Lambda %s: %w", functionName, err))
		}
	}

	// Cleanup OIDC providers
	for _, providerARN := range t.oidcProviders {
		if err := t.cleanupOIDCProvider(ctx, awsHelper, providerARN); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup OIDC provider %s: %w", providerARN, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// cleanupLambdaFunction deletes a Lambda function with retries
func (t *ResourceTracker) cleanupLambdaFunction(ctx context.Context, awsHelper *AWSTestHelper, functionName string) error {
	// Try to delete the function, ignore if it doesn't exist
	_, err := awsHelper.LambdaClient.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		// Ignore ResourceNotFoundException - function already deleted
		if isResourceNotFound(err) {
			return nil
		}
		return err
	}

	// Wait for function to be fully deleted
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		_, err := awsHelper.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})

		if err != nil && isResourceNotFound(err) {
			return nil // Successfully deleted
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("function %s still exists after deletion attempt", functionName)
}

// isResourceNotFound checks if the error is a ResourceNotFoundException
func isResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "ResourceNotFoundException")
}

// cleanupOIDCProvider deletes an IAM OIDC provider
func (t *ResourceTracker) cleanupOIDCProvider(ctx context.Context, awsHelper *AWSTestHelper, providerARN string) error {
	_, err := awsHelper.IAMClient.DeleteOpenIDConnectProvider(ctx, &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(providerARN),
	})
	if err != nil {
		if isEntityNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// isEntityNotFound checks if the error is an entity not found error
func isEntityNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "NoSuchEntity")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
