package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSTestHelper provides direct AWS API access for test verification
type AWSTestHelper struct {
	LambdaClient *lambda.Client
	ECRClient    *ecr.Client
	S3Client     *s3.Client
	IAMClient    *iam.Client
	Region       string
}

// NewAWSTestHelper creates a new AWS test helper using default credentials
// It respects AWS_PROFILE environment variable
func NewAWSTestHelper(ctx context.Context) (*AWSTestHelper, error) {
	// Load AWS config using default credential chain (respects AWS_PROFILE)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSTestHelper{
		LambdaClient: lambda.NewFromConfig(cfg),
		ECRClient:    ecr.NewFromConfig(cfg),
		S3Client:     s3.NewFromConfig(cfg),
		IAMClient:    iam.NewFromConfig(cfg),
		Region:       cfg.Region,
	}, nil
}

// VerifyFunctionExists checks if a Lambda function exists
func (h *AWSTestHelper) VerifyFunctionExists(ctx context.Context, functionName string) (bool, error) {
	_, err := h.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		// If function not found, return false without error
		if isResourceNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// VerifyFunctionDeleted verifies that a Lambda function has been deleted
func (h *AWSTestHelper) VerifyFunctionDeleted(ctx context.Context, functionName string) error {
	exists, err := h.VerifyFunctionExists(ctx, functionName)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("function %s still exists", functionName)
	}

	return nil
}

// WaitForFunctionActive waits for a Lambda function to reach Active state
func (h *AWSTestHelper) WaitForFunctionActive(ctx context.Context, functionName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		output, err := h.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})

		if err != nil {
			return fmt.Errorf("failed to get function: %w", err)
		}

		state := output.Configuration.State
		if state == "Active" {
			return nil
		}

		if state == "Failed" {
			return fmt.Errorf("function entered Failed state: %s", aws.ToString(output.Configuration.StateReason))
		}

		// Wait before retrying
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for function to become active")
}

// WaitForFunctionReady waits for a Lambda function to be ready for updates
// This checks both State=Active and LastUpdateStatus=Successful, and ensures
// the function remains stable for a few seconds to avoid race conditions with
// AWS Lambda's background processing (e.g., creating execution roles, setting up VPC, etc.)
func (h *AWSTestHelper) WaitForFunctionReady(ctx context.Context, functionName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var firstReadyTime time.Time
	// Function must stay in ready state for this long to ensure AWS background processing is complete
	// Increase this if you still see "ResourceConflictException: An update is in progress" errors
	stabilityPeriod := 5 * time.Second

	for time.Now().Before(deadline) {
		output, err := h.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})

		if err != nil {
			return fmt.Errorf("failed to get function: %w", err)
		}

		state := string(output.Configuration.State)
		lastUpdateStatus := string(output.Configuration.LastUpdateStatus)

		// Check if function is in a failed state
		if state == "Failed" {
			return fmt.Errorf("function entered Failed state: %s", aws.ToString(output.Configuration.StateReason))
		}

		if lastUpdateStatus == "Failed" {
			return fmt.Errorf("function update failed: %s", aws.ToString(output.Configuration.LastUpdateStatusReason))
		}

		// Function is ready when it's Active and last update is Successful
		if state == "Active" && lastUpdateStatus == "Successful" {
			// First time seeing the ready state
			if firstReadyTime.IsZero() {
				firstReadyTime = time.Now()
			}

			// Check if function has been stable for the required period
			if time.Since(firstReadyTime) >= stabilityPeriod {
				return nil
			}
		} else {
			// Reset if state changes
			firstReadyTime = time.Time{}
		}

		// Wait before retrying
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for function to be ready (current state may still be updating)")
}

// GetFunctionRuntime returns the runtime of a Lambda function
func (h *AWSTestHelper) GetFunctionRuntime(ctx context.Context, functionName string) (string, error) {
	output, err := h.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		return "", fmt.Errorf("failed to get function: %w", err)
	}

	return string(output.Configuration.Runtime), nil
}

// ListFunctions lists all Lambda functions (for verification)
func (h *AWSTestHelper) ListFunctions(ctx context.Context) ([]string, error) {
	var functionNames []string

	paginator := lambda.NewListFunctionsPaginator(h.LambdaClient, &lambda.ListFunctionsInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions: %w", err)
		}

		for _, fn := range output.Functions {
			functionNames = append(functionNames, aws.ToString(fn.FunctionName))
		}
	}

	return functionNames, nil
}

// VerifyECRRepositoryExists checks if an ECR repository exists
func (h *AWSTestHelper) VerifyECRRepositoryExists(ctx context.Context, repositoryName string) (bool, error) {
	_, err := h.ECRClient.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repositoryName},
	})

	if err != nil {
		// If repository not found, return false without error
		if isRepositoryNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetFunctionVersions returns all versions of a Lambda function
func (h *AWSTestHelper) GetFunctionVersions(ctx context.Context, functionName string) ([]string, error) {
	var versions []string

	paginator := lambda.NewListVersionsByFunctionPaginator(h.LambdaClient, &lambda.ListVersionsByFunctionInput{
		FunctionName: aws.String(functionName),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list versions: %w", err)
		}

		for _, version := range output.Versions {
			versions = append(versions, aws.ToString(version.Version))
		}
	}

	return versions, nil
}

// isResourceNotFoundError checks if the error indicates a resource was not found
func isResourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsString(errStr, "ResourceNotFoundException") ||
		containsString(errStr, "ResourceNotFound") ||
		containsString(errStr, "does not exist")
}

// isRepositoryNotFoundError checks if the error indicates a repository was not found
func isRepositoryNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsString(errStr, "RepositoryNotFoundException") ||
		containsString(errStr, "RepositoryNotFound")
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetFunctionEnvironment returns the environment variables of a Lambda function
func (h *AWSTestHelper) GetFunctionEnvironment(ctx context.Context, functionName string) (map[string]string, error) {
	output, err := h.LambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	if output.Configuration == nil || output.Configuration.Environment == nil {
		return make(map[string]string), nil
	}

	return output.Configuration.Environment.Variables, nil
}

// VerifyS3BucketExists checks if an S3 bucket exists
func (h *AWSTestHelper) VerifyS3BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := h.S3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Check if it's a not found error
		errStr := err.Error()
		if containsString(errStr, "NotFound") ||
			containsString(errStr, "NoSuchBucket") ||
			containsString(errStr, "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// VerifyOIDCDiscoveryExists checks if OIDC discovery documents exist in S3 bucket
func (h *AWSTestHelper) VerifyOIDCDiscoveryExists(ctx context.Context, bucketName string) (bool, error) {
	// Check for .well-known/openid-configuration
	_, err := h.S3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(".well-known/openid-configuration"),
	})
	if err != nil {
		return false, nil
	}

	// Check for keys.json
	_, err = h.S3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("keys.json"),
	})
	if err != nil {
		return false, nil
	}

	return true, nil
}

// VerifyOIDCProviderExists checks if an IAM OIDC provider exists for the given issuer URL
func (h *AWSTestHelper) VerifyOIDCProviderExists(ctx context.Context, issuerURL string) (bool, error) {
	listOutput, err := h.IAMClient.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return false, fmt.Errorf("failed to list OIDC providers: %w", err)
	}

	for _, provider := range listOutput.OpenIDConnectProviderList {
		providerARN := aws.ToString(provider.Arn)

		getOutput, err := h.IAMClient.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: aws.String(providerARN),
		})
		if err != nil {
			continue
		}

		// Check both with and without https:// prefix
		providerURL := aws.ToString(getOutput.Url)
		if providerURL == issuerURL || providerURL == "https://"+issuerURL || "https://"+providerURL == issuerURL {
			return true, nil
		}
	}

	return false, nil
}
