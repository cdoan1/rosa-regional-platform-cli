package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type FunctionInfo struct {
	Name          string `json:"name"`
	Created       string `json:"created"`
	Runtime       string `json:"runtime"`
	ARN           string `json:"arn"`
	LatestVersion string `json:"latest_version"`
}

func (c *Client) ListFunctions(ctx context.Context) ([]FunctionInfo, error) {
	var functions []FunctionInfo
	var nextMarker *string

	for {
		input := &lambda.ListFunctionsInput{
			Marker: nextMarker,
		}

		output, err := c.lambda.ListFunctions(ctx, input)
		if err != nil {
			return nil, &LambdaError{
				Operation: "list",
				Message:   fmt.Sprintf("failed to list Lambda functions: %v", err),
			}
		}

		for _, fn := range output.Functions {
			functionName := aws.ToString(fn.FunctionName)

			// Get latest published version
			latestVersion := c.getLatestPublishedVersion(ctx, functionName)

			functions = append(functions, FunctionInfo{
				Name:          functionName,
				Created:       aws.ToString(fn.LastModified),
				Runtime:       string(fn.Runtime),
				ARN:           aws.ToString(fn.FunctionArn),
				LatestVersion: latestVersion,
			})
		}

		if output.NextMarker == nil {
			break
		}
		nextMarker = output.NextMarker
	}

	return functions, nil
}

// getLatestPublishedVersion returns the latest published version number or "-" if none
func (c *Client) getLatestPublishedVersion(ctx context.Context, functionName string) string {
	// List versions for the function
	output, err := c.lambda.ListVersionsByFunction(ctx, &lambda.ListVersionsByFunctionInput{
		FunctionName: aws.String(functionName),
		MaxItems:     aws.Int32(10), // Get last 10 versions to find the latest
	})
	if err != nil {
		return "-"
	}

	// Find the highest version number (excluding $LATEST)
	latestVersion := "-"
	for _, v := range output.Versions {
		version := aws.ToString(v.Version)
		if version != "$LATEST" {
			latestVersion = version
		}
	}

	return latestVersion
}
