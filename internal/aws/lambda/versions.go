package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type VersionInfo struct {
	Version      string `json:"version"`
	Description  string `json:"description"`
	LastModified string `json:"last_modified"`
}

func (c *Client) ListVersions(ctx context.Context, functionName string) ([]VersionInfo, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return nil, err
	} else if !exists {
		return nil, &LambdaError{
			Operation: "list versions",
			Message:   fmt.Sprintf("Lambda function '%s' not found", functionName),
		}
	}

	var versions []VersionInfo
	var nextMarker *string

	for {
		input := &lambda.ListVersionsByFunctionInput{
			FunctionName: aws.String(functionName),
			Marker:       nextMarker,
		}

		output, err := c.lambda.ListVersionsByFunction(ctx, input)
		if err != nil {
			return nil, &LambdaError{
				Operation: "list versions",
				Message:   fmt.Sprintf("failed to list versions: %v", err),
			}
		}

		for _, v := range output.Versions {
			// Skip $LATEST
			if aws.ToString(v.Version) == "$LATEST" {
				continue
			}

			versions = append(versions, VersionInfo{
				Version:      aws.ToString(v.Version),
				Description:  aws.ToString(v.Description),
				LastModified: aws.ToString(v.LastModified),
			})
		}

		if output.NextMarker == nil {
			break
		}
		nextMarker = output.NextMarker
	}

	return versions, nil
}
