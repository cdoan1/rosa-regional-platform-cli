package lambda

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const (
	executionRoleName     = "rosactl-lambda-execution-role"
	oidcExecutionRoleName = "rosactl-lambda-oidc-execution-role"
	lambdaTrustPolicy     = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`
	basicExecutionPolicyARN = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
	oidcPolicyDocument      = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:CreateBucket",
        "s3:ListBucket",
        "s3:PutObject",
        "s3:PutBucketPolicy",
        "s3:PutPublicAccessBlock",
        "s3:PutBucketPublicAccessBlock",
        "s3:DeleteObject",
        "s3:DeleteBucket",
        "s3:GetBucketLocation"
      ],
      "Resource": [
        "arn:aws:s3:::oidc-issuer-*",
        "arn:aws:s3:::oidc-issuer-*/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreateOpenIDConnectProvider",
        "iam:GetOpenIDConnectProvider",
        "iam:ListOpenIDConnectProviders",
        "iam:DeleteOpenIDConnectProvider"
      ],
      "Resource": "*"
    }
  ]
}`
)

func (c *Client) ensureExecutionRole(ctx context.Context) (string, error) {
	roleARN, err := c.getRole(ctx, executionRoleName)
	if err == nil {
		return roleARN, nil
	}

	if err := c.createRole(ctx); err != nil {
		return "", err
	}

	if err := c.attachPolicy(ctx); err != nil {
		return "", err
	}

	time.Sleep(10 * time.Second)

	roleARN, err = c.getRole(ctx, executionRoleName)
	if err != nil {
		return "", fmt.Errorf("failed to verify role creation: %w", err)
	}

	return roleARN, nil
}

func (c *Client) getRole(ctx context.Context, roleName string) (string, error) {
	output, err := c.iam.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(output.Role.Arn), nil
}

func (c *Client) createRole(ctx context.Context) error {
	_, err := c.iam.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(executionRoleName),
		AssumeRolePolicyDocument: aws.String(lambdaTrustPolicy),
		Description:              aws.String("Execution role for rosactl Lambda functions"),
	})
	if err != nil {
		var alreadyExists *types.EntityAlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return nil
		}
		return fmt.Errorf("failed to create IAM role: %w", err)
	}

	return nil
}

func (c *Client) attachPolicy(ctx context.Context) error {
	_, err := c.iam.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(executionRoleName),
		PolicyArn: aws.String(basicExecutionPolicyARN),
	})
	if err != nil {
		return fmt.Errorf("failed to attach policy: %w", err)
	}

	return nil
}

// ensureOIDCExecutionRole ensures the OIDC Lambda execution role exists with S3 and IAM permissions
func (c *Client) ensureOIDCExecutionRole(ctx context.Context) (string, error) {
	roleARN, err := c.getRole(ctx, oidcExecutionRoleName)
	if err == nil {
		return roleARN, nil
	}

	if err := c.createOIDCRole(ctx); err != nil {
		return "", err
	}

	// Attach CloudWatch Logs policy
	if err := c.attachPolicyToRole(ctx, oidcExecutionRoleName, basicExecutionPolicyARN); err != nil {
		return "", err
	}

	// Attach inline OIDC policy
	if err := c.attachOIDCInlinePolicy(ctx); err != nil {
		return "", err
	}

	time.Sleep(10 * time.Second) // IAM eventual consistency

	roleARN, err = c.getRole(ctx, oidcExecutionRoleName)
	if err != nil {
		return "", fmt.Errorf("failed to verify role creation: %w", err)
	}

	return roleARN, nil
}

func (c *Client) createOIDCRole(ctx context.Context) error {
	_, err := c.iam.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(oidcExecutionRoleName),
		AssumeRolePolicyDocument: aws.String(lambdaTrustPolicy),
		Description:              aws.String("Execution role for rosactl OIDC Lambda function"),
	})
	if err != nil {
		var alreadyExists *types.EntityAlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return nil
		}
		return fmt.Errorf("failed to create IAM role: %w", err)
	}
	return nil
}

func (c *Client) attachOIDCInlinePolicy(ctx context.Context) error {
	_, err := c.iam.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(oidcExecutionRoleName),
		PolicyName:     aws.String("OIDCProviderManagement"),
		PolicyDocument: aws.String(oidcPolicyDocument),
	})
	if err != nil {
		return fmt.Errorf("failed to attach OIDC inline policy: %w", err)
	}
	return nil
}

func (c *Client) attachPolicyToRole(ctx context.Context, roleName, policyARN string) error {
	_, err := c.iam.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyARN),
	})
	if err != nil {
		return fmt.Errorf("failed to attach policy: %w", err)
	}
	return nil
}
