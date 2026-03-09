package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func (c *Client) DeleteFunction(ctx context.Context, functionName string) error {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return err
	} else if !exists {
		return &LambdaError{
			Operation: "delete",
			Message:   fmt.Sprintf("Lambda function '%s' not found", functionName),
		}
	}

	_, err := c.lambda.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return &LambdaError{
			Operation: "delete",
			Message:   fmt.Sprintf("failed to delete Lambda function: %v", err),
		}
	}

	return nil
}
