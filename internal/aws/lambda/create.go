package lambda

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/crypto"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/python"
)

func (c *Client) CreateFunction(ctx context.Context, functionName string) (string, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return "", err
	} else if exists {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("Lambda function '%s' already exists", functionName),
		}
	}

	roleARN, err := c.ensureExecutionRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to ensure execution role: %w", err)
	}

	zipBytes, err := python.PackageHandler(python.DefaultHandler)
	if err != nil {
		return "", fmt.Errorf("failed to package Python handler: %w", err)
	}

	// Set creation timestamp
	createdAt := time.Now().UTC().Format(time.RFC3339)

	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Runtime:      types.RuntimePython312,
		Role:         aws.String(roleARN),
		Handler:      aws.String("lambda_function.lambda_handler"),
		Code: &types.FunctionCode{
			ZipFile: zipBytes,
		},
		Timeout:     aws.Int32(30),
		MemorySize:  aws.Int32(128),
		Description: aws.String("Created by rosactl"),
		Environment: &types.Environment{
			Variables: map[string]string{
				"CREATED_AT": createdAt,
			},
		},
	}

	output, err := c.lambda.CreateFunction(ctx, input)
	if err != nil {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("failed to create Lambda function: %v", err),
		}
	}

	waiter := lambda.NewFunctionActiveV2Waiter(c.lambda)
	if err := waiter.Wait(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	}, 2*time.Minute); err != nil {
		return "", fmt.Errorf("timeout waiting for function to become active: %w", err)
	}

	// Publish initial version
	versionOutput, err := c.lambda.PublishVersion(ctx, &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("Initial version"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish initial version: %w", err)
	}

	return fmt.Sprintf("%s (version: %s)", aws.ToString(output.FunctionArn), aws.ToString(versionOutput.Version)), nil
}

func (c *Client) CreateOIDCFunction(ctx context.Context, functionName string) (string, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return "", err
	} else if exists {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("Lambda function '%s' already exists", functionName),
		}
	}

	// Generate RSA key pair for this Lambda function
	fmt.Println("Generating RSA key pair for OIDC Lambda...")
	keyPair, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA key pair: %w", err)
	}
	fmt.Printf("Generated RSA key pair (kid: %s)\n", keyPair.KID)

	// Save private key to temp directory
	privateKeyPEM := crypto.PrivateKeyToPEM(keyPair.PrivateKey)
	keyFileName := fmt.Sprintf("oidc-private-key-%s.pem", keyPair.KID)
	keyFilePath := filepath.Join(os.TempDir(), keyFileName)

	if err := os.WriteFile(keyFilePath, privateKeyPEM, 0600); err != nil {
		return "", fmt.Errorf("failed to save private key: %w", err)
	}

	fmt.Printf("\n⚠️  Private key saved to: %s\n", keyFilePath)
	fmt.Println("⚠️  Keep this file secure! It can be used to sign JWTs for this OIDC issuer.")
	fmt.Println()

	roleARN, err := c.ensureOIDCExecutionRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to ensure OIDC execution role: %w", err)
	}

	zipBytes, err := python.PackageHandler(python.OIDCHandler)
	if err != nil {
		return "", fmt.Errorf("failed to package Python OIDC handler: %w", err)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)

	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Runtime:      types.RuntimePython312,
		Role:         aws.String(roleARN),
		Handler:      aws.String("lambda_function.lambda_handler"),
		Code: &types.FunctionCode{
			ZipFile: zipBytes,
		},
		Timeout:     aws.Int32(60),  // OIDC creation takes longer
		MemorySize:  aws.Int32(256), // More memory for S3/IAM operations
		Description: aws.String("OIDC provider manager created by rosactl"),
		Environment: &types.Environment{
			Variables: map[string]string{
				"CREATED_AT": createdAt,
				"JWK_N":      keyPair.N,   // RSA modulus (base64url)
				"JWK_E":      keyPair.E,   // RSA exponent (base64url)
				"JWK_KID":    keyPair.KID, // Key ID
			},
		},
	}

	output, err := c.lambda.CreateFunction(ctx, input)
	if err != nil {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("failed to create Lambda function: %v", err),
		}
	}

	waiter := lambda.NewFunctionActiveV2Waiter(c.lambda)
	if err := waiter.Wait(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	}, 2*time.Minute); err != nil {
		return "", fmt.Errorf("timeout waiting for function to become active: %w", err)
	}

	versionOutput, err := c.lambda.PublishVersion(ctx, &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("Initial version"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish initial version: %w", err)
	}

	return fmt.Sprintf("%s (version: %s)", aws.ToString(output.FunctionArn), aws.ToString(versionOutput.Version)), nil
}

func (c *Client) CreateOIDCDeleteFunction(ctx context.Context, functionName string) (string, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return "", err
	} else if exists {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("Lambda function '%s' already exists", functionName),
		}
	}

	roleARN, err := c.ensureOIDCExecutionRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to ensure OIDC execution role: %w", err)
	}

	zipBytes, err := python.PackageHandler(python.OIDCDeleteHandler)
	if err != nil {
		return "", fmt.Errorf("failed to package Python OIDC delete handler: %w", err)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)

	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Runtime:      types.RuntimePython312,
		Role:         aws.String(roleARN),
		Handler:      aws.String("lambda_function.lambda_handler"),
		Code: &types.FunctionCode{
			ZipFile: zipBytes,
		},
		Timeout:     aws.Int32(60), // OIDC deletion takes time
		MemorySize:  aws.Int32(256),
		Description: aws.String("OIDC issuer deletion manager created by rosactl"),
		Environment: &types.Environment{
			Variables: map[string]string{
				"CREATED_AT": createdAt,
			},
		},
	}

	output, err := c.lambda.CreateFunction(ctx, input)
	if err != nil {
		return "", &LambdaError{
			Operation: "create",
			Message:   fmt.Sprintf("failed to create Lambda function: %v", err),
		}
	}

	waiter := lambda.NewFunctionActiveV2Waiter(c.lambda)
	if err := waiter.Wait(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	}, 2*time.Minute); err != nil {
		return "", fmt.Errorf("timeout waiting for function to become active: %w", err)
	}

	versionOutput, err := c.lambda.PublishVersion(ctx, &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("Initial version"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish initial version: %w", err)
	}

	return fmt.Sprintf("%s (version: %s)", aws.ToString(output.FunctionArn), aws.ToString(versionOutput.Version)), nil
}

func (c *Client) functionExists(ctx context.Context, functionName string) (bool, error) {
	_, err := c.lambda.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
