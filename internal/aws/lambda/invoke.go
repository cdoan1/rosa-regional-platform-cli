package lambda

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func (c *Client) InvokeFunction(ctx context.Context, functionName string, version string) (string, string, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return "", "", err
	} else if !exists {
		return "", "", &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("Lambda function '%s' not found", functionName),
		}
	}

	// Build function name with version qualifier if specified
	qualifiedName := functionName
	if version != "" {
		qualifiedName = fmt.Sprintf("%s:%s", functionName, version)
	}

	output, err := c.lambda.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(qualifiedName),
		InvocationType: types.InvocationTypeRequestResponse,
		LogType:        types.LogTypeTail,
	})
	if err != nil {
		return "", "", &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("failed to invoke Lambda function: %v", err),
		}
	}

	if output.FunctionError != nil {
		return "", "", &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("Lambda function error: %s", string(output.Payload)),
		}
	}

	logs := ""
	if output.LogResult != nil {
		logBytes, err := base64.StdEncoding.DecodeString(*output.LogResult)
		if err == nil {
			logs = string(logBytes)
		}
	}

	// Parse and format the response payload
	result := formatPayload(output.Payload)

	return result, logs, nil
}

// InvokeFunctionWithPayload invokes a Lambda function with a custom JSON payload
func (c *Client) InvokeFunctionWithPayload(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	if exists, err := c.functionExists(ctx, functionName); err != nil {
		return nil, err
	} else if !exists {
		return nil, &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("Lambda function '%s' not found", functionName),
		}
	}

	output, err := c.lambda.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
		LogType:        types.LogTypeTail,
	})
	if err != nil {
		return nil, &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("failed to invoke Lambda function: %v", err),
		}
	}

	if output.FunctionError != nil {
		return nil, &LambdaError{
			Operation: "invoke",
			Message:   fmt.Sprintf("Lambda function error: %s", string(output.Payload)),
		}
	}

	return output.Payload, nil
}

// formatPayload attempts to parse and pretty-print the Lambda response
func formatPayload(payload []byte) string {
	// Try to parse as JSON
	var response map[string]interface{}
	if err := json.Unmarshal(payload, &response); err != nil {
		// If not JSON, return as-is
		return string(payload)
	}

	// Check if there's a "body" field that's a JSON string
	if bodyStr, ok := response["body"].(string); ok {
		var bodyObj interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyObj); err == nil {
			// Successfully parsed body as JSON, replace it with the parsed object
			response["body"] = bodyObj
		}
	}

	// Pretty-print the response
	formatted, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return string(payload)
	}

	return string(formatted)
}
