package lambda

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/python"
)

func (c *Client) UpdateFunctionCode(ctx context.Context, functionName string) (string, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return "", err
	} else if !exists {
		return "", &LambdaError{
			Operation: "update",
			Message:   fmt.Sprintf("Lambda function '%s' not found", functionName),
		}
	}

	// Get current function configuration
	getFuncOutput, err := c.lambda.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get function details: %w", err)
	}

	// Use datetime handler
	handlerCode := python.DefaultHandler

	// Preserve existing CREATED_AT if it exists, otherwise get it from version 1
	var createdAt string
	if existingEnv := getFuncOutput.Configuration.Environment; existingEnv != nil {
		if existingCreatedAt, exists := existingEnv.Variables["CREATED_AT"]; exists {
			createdAt = existingCreatedAt
		}
	}

	// If no CREATED_AT exists in environment, try to get it from version 1
	if createdAt == "" {
		version1, err := c.lambda.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
			Qualifier:    aws.String("1"),
		})
		if err == nil && version1.Configuration != nil {
			createdAt = aws.ToString(version1.Configuration.LastModified)
		} else {
			// Fallback to current LastModified if version 1 doesn't exist
			createdAt = aws.ToString(getFuncOutput.Configuration.LastModified)
		}
	}

	envVars := map[string]string{
		"CREATED_AT": createdAt,
	}

	// Package new handler
	zipBytes, err := python.PackageHandler(handlerCode)
	if err != nil {
		return "", fmt.Errorf("failed to package Python handler: %w", err)
	}

	// Update function code
	_, err = c.lambda.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(functionName),
		ZipFile:      zipBytes,
	})
	if err != nil {
		return "", &LambdaError{
			Operation: "update",
			Message:   fmt.Sprintf("failed to update function code: %v", err),
		}
	}

	// Update environment variables
	_, err = c.lambda.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(functionName),
		Environment: &types.Environment{
			Variables: envVars,
		},
	})
	if err != nil {
		return "", &LambdaError{
			Operation: "update",
			Message:   fmt.Sprintf("failed to update function configuration: %v", err),
		}
	}

	// Wait for update to complete
	waiter := lambda.NewFunctionUpdatedV2Waiter(c.lambda)
	if err := waiter.Wait(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	}, 2*time.Minute); err != nil {
		return "", fmt.Errorf("timeout waiting for function update: %w", err)
	}

	// Publish new version
	versionOutput, err := c.lambda.PublishVersion(ctx, &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("Updated to datetime handler"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish version: %w", err)
	}

	return aws.ToString(versionOutput.Version), nil
}
