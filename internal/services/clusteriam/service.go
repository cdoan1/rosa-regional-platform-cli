package clusteriam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/cloudformation/templates"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/crypto"
)

type CreateIAMRequest struct {
	ClusterName    string
	OIDCIssuerURL  string
	OIDCThumbprint string // Optional - will be fetched if not provided
	AWSConfig      aws.Config
}

type CreateIAMResponse struct {
	StackID string
	Outputs map[string]string
}

type DeleteIAMRequest struct {
	ClusterName string
	AWSConfig   aws.Config
}

// CreateIAM creates cluster IAM resources via CloudFormation
func CreateIAM(ctx context.Context, req *CreateIAMRequest) (*CreateIAMResponse, error) {
	// Validate OIDC issuer URL
	if !strings.HasPrefix(req.OIDCIssuerURL, "https://") {
		return nil, fmt.Errorf("OIDC issuer URL must start with https://")
	}

	// Fetch TLS thumbprint if not provided
	thumbprint := req.OIDCThumbprint
	if thumbprint == "" {
		var err error
		thumbprint, err = crypto.GetOIDCThumbprint(ctx, req.OIDCIssuerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch TLS thumbprint: %w", err)
		}
	}

	// Derive OIDC issuer domain
	oidcIssuerDomain, err := crypto.GetOIDCIssuerDomain(req.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OIDC issuer URL: %w", err)
	}

	// Read CloudFormation template
	templateBody, err := templates.Read("cluster-iam.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(req.AWSConfig)

	// Prepare stack parameters
	stackName := fmt.Sprintf("rosa-%s-iam", req.ClusterName)
	params := &cloudformation.CreateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":      req.ClusterName,
			"OIDCIssuerURL":    req.OIDCIssuerURL,
			"OIDCIssuerDomain": oidcIssuerDomain,
			"OIDCThumbprint":   thumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("Cluster"),
				Value: aws.String(req.ClusterName),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("rosactl"),
			},
			{
				Key:   aws.String("red-hat-managed"),
				Value: aws.String("true"),
			},
		},
		WaitTimeout: 15 * time.Minute,
	}

	// Create stack
	output, err := cfnClient.CreateStack(ctx, params)
	if err != nil {
		// Check if stack already exists, try update instead
		var alreadyExistsErr *cloudformation.StackAlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			return updateIAM(ctx, cfnClient, req, stackName, templateBody, oidcIssuerDomain, thumbprint)
		}
		return nil, fmt.Errorf("failed to create stack: %w", err)
	}

	return &CreateIAMResponse{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

func updateIAM(ctx context.Context, cfnClient *cloudformation.Client, req *CreateIAMRequest, stackName, templateBody, oidcIssuerDomain, thumbprint string) (*CreateIAMResponse, error) {
	params := &cloudformation.UpdateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":      req.ClusterName,
			"OIDCIssuerURL":    req.OIDCIssuerURL,
			"OIDCIssuerDomain": oidcIssuerDomain,
			"OIDCThumbprint":   thumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		WaitTimeout: 15 * time.Minute,
	}

	output, err := cfnClient.UpdateStack(ctx, params)
	if err != nil {
		var noChanges *cloudformation.NoChangesError
		if errors.As(err, &noChanges) {
			current, descErr := cfnClient.GetStackOutputs(ctx, stackName)
			if descErr != nil {
				return nil, descErr
			}
			return &CreateIAMResponse{
				StackID: current.StackID,
				Outputs: current.Outputs,
			}, nil
		}
		return nil, fmt.Errorf("failed to update stack: %w", err)
	}

	return &CreateIAMResponse{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

// DeleteIAM deletes cluster IAM resources
func DeleteIAM(ctx context.Context, req *DeleteIAMRequest) error {
	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(req.AWSConfig)

	// Delete stack
	stackName := fmt.Sprintf("rosa-%s-iam", req.ClusterName)
	err := cfnClient.DeleteStack(ctx, stackName, 15*time.Minute)
	if err != nil {
		var notFound *cloudformation.StackNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("failed to delete stack: %w", err)
	}

	return nil
}
