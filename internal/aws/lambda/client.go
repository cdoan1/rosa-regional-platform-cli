package lambda

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	awsconfig "github.com/openshift-online/rosa-regional-platform-cli/internal/aws"
)

type Client struct {
	lambda *lambda.Client
	iam    *iam.Client
	cfg    aws.Config
}

func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := awsconfig.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		lambda: lambda.NewFromConfig(cfg),
		iam:    iam.NewFromConfig(cfg),
		cfg:    cfg,
	}, nil
}

// functionExists checks if a Lambda function exists
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
