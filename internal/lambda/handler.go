package lambda

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
)

// Event represents the Lambda function input
type Event struct {
	Action         string `json:"action"`
	ClusterName    string `json:"cluster_name"`
	OIDCIssuerURL  string `json:"oidc_issuer_url"`
	OIDCThumbprint string `json:"oidc_thumbprint"`
	// VPC parameters
	VpcCidr            string   `json:"vpc_cidr,omitempty"`
	PublicSubnetCidrs  []string `json:"public_subnet_cidrs,omitempty"`
	PrivateSubnetCidrs []string `json:"private_subnet_cidrs,omitempty"`
	AvailabilityZones  []string `json:"availability_zones,omitempty"`
	SingleNatGateway   bool     `json:"single_nat_gateway,omitempty"`
}

// Response represents the Lambda function output
type Response struct {
	StackID string            `json:"stack_id"`
	Outputs map[string]string `json:"outputs"`
	Error   string            `json:"error,omitempty"`
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, event Event) (Response, error) {
	fmt.Printf("Received event: %+v\n", event)

	switch event.Action {
	case "apply-cluster-iam":
		return applyClusterIAM(ctx, event)
	case "delete-cluster-iam":
		return deleteClusterIAM(ctx, event)
	case "apply-cluster-vpc":
		return applyClusterVPC(ctx, event)
	case "delete-cluster-vpc":
		return deleteClusterVPC(ctx, event)
	default:
		return Response{
			Error: fmt.Sprintf("unknown action: %s", event.Action),
		}, fmt.Errorf("unknown action: %s", event.Action)
	}
}

// applyClusterIAM applies the cluster IAM CloudFormation template
func applyClusterIAM(ctx context.Context, event Event) (Response, error) {
	if event.ClusterName == "" {
		return Response{Error: "cluster_name is required"}, fmt.Errorf("cluster_name is required")
	}
	if event.OIDCIssuerURL == "" {
		return Response{Error: "oidc_issuer_url is required"}, fmt.Errorf("oidc_issuer_url is required")
	}
	if event.OIDCThumbprint == "" {
		return Response{Error: "oidc_thumbprint is required"}, fmt.Errorf("oidc_thumbprint is required")
	}

	fmt.Println("Applying cluster IAM CloudFormation template...")

	// Read template file
	templateBody, err := readTemplateFile("cluster-iam.yaml")
	if err != nil {
		return Response{Error: err.Error()}, err
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to load AWS config: %v", err)}, err
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Prepare stack parameters
	stackName := fmt.Sprintf("rosa-%s-iam", event.ClusterName)
	params := &cloudformation.CreateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":     event.ClusterName,
			"OIDCIssuerURL":   event.OIDCIssuerURL,
			"OIDCThumbprint":  event.OIDCThumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		Tags: []types.Tag{
			{
				Key:   stringPtr("Cluster"),
				Value: stringPtr(event.ClusterName),
			},
			{
				Key:   stringPtr("ManagedBy"),
				Value: stringPtr("rosactl"),
			},
			{
				Key:   stringPtr("red-hat-managed"),
				Value: stringPtr("true"),
			},
		},
		WaitTimeout: 15 * time.Minute,
	}

	fmt.Printf("Creating CloudFormation stack: %s\n", stackName)

	// Create stack
	output, err := cfnClient.CreateStack(ctx, params)
	if err != nil {
		// Check if stack already exists, try update instead
		if isAlreadyExistsError(err) {
			fmt.Printf("Stack already exists, attempting update...\n")
			return updateClusterIAM(ctx, cfnClient, event, stackName, templateBody)
		}
		return Response{Error: fmt.Sprintf("failed to create stack: %v", err)}, err
	}

	fmt.Printf("Stack created successfully: %s\n", output.StackID)

	return Response{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

// updateClusterIAM updates an existing cluster IAM CloudFormation stack
func updateClusterIAM(ctx context.Context, cfnClient *cloudformation.Client, event Event, stackName, templateBody string) (Response, error) {
	params := &cloudformation.UpdateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ClusterName":     event.ClusterName,
			"OIDCIssuerURL":   event.OIDCIssuerURL,
			"OIDCThumbprint":  event.OIDCThumbprint,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		WaitTimeout: 15 * time.Minute,
	}

	output, err := cfnClient.UpdateStack(ctx, params)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to update stack: %v", err)}, err
	}

	fmt.Printf("Stack updated successfully: %s\n", output.StackID)

	return Response{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

// deleteClusterIAM deletes the cluster IAM CloudFormation stack
func deleteClusterIAM(ctx context.Context, event Event) (Response, error) {
	if event.ClusterName == "" {
		return Response{Error: "cluster_name is required"}, fmt.Errorf("cluster_name is required")
	}

	fmt.Printf("Deleting cluster IAM CloudFormation stack for: %s\n", event.ClusterName)

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to load AWS config: %v", err)}, err
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Delete stack
	stackName := fmt.Sprintf("rosa-%s-iam", event.ClusterName)
	err = cfnClient.DeleteStack(ctx, stackName, 15*time.Minute)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to delete stack: %v", err)}, err
	}

	fmt.Printf("Stack deleted successfully: %s\n", stackName)

	return Response{
		StackID: stackName,
		Outputs: map[string]string{"status": "deleted"},
	}, nil
}

// applyClusterVPC applies the cluster VPC CloudFormation template
func applyClusterVPC(ctx context.Context, event Event) (Response, error) {
	if event.ClusterName == "" {
		return Response{Error: "cluster_name is required"}, fmt.Errorf("cluster_name is required")
	}

	fmt.Println("Applying cluster VPC CloudFormation template...")

	// Read template file
	templateBody, err := readTemplateFile("cluster-vpc.yaml")
	if err != nil {
		return Response{Error: err.Error()}, err
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to load AWS config: %v", err)}, err
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Prepare stack parameters
	stackName := fmt.Sprintf("rosa-%s-vpc", event.ClusterName)
	params := map[string]string{
		"ClusterName":       event.ClusterName,
		"SingleNatGateway":  fmt.Sprintf("%t", event.SingleNatGateway),
	}

	// Add optional parameters
	if event.VpcCidr != "" {
		params["VpcCidr"] = event.VpcCidr
	}
	if len(event.PublicSubnetCidrs) > 0 {
		params["PublicSubnetCidrs"] = strings.Join(event.PublicSubnetCidrs, ",")
	}
	if len(event.PrivateSubnetCidrs) > 0 {
		params["PrivateSubnetCidrs"] = strings.Join(event.PrivateSubnetCidrs, ",")
	}
	if len(event.AvailabilityZones) >= 1 {
		params["AvailabilityZone1"] = event.AvailabilityZones[0]
	}
	if len(event.AvailabilityZones) >= 2 {
		params["AvailabilityZone2"] = event.AvailabilityZones[1]
	}
	if len(event.AvailabilityZones) >= 3 {
		params["AvailabilityZone3"] = event.AvailabilityZones[2]
	}

	createParams := &cloudformation.CreateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters:   params,
		Tags: []types.Tag{
			{
				Key:   stringPtr("Cluster"),
				Value: stringPtr(event.ClusterName),
			},
			{
				Key:   stringPtr("ManagedBy"),
				Value: stringPtr("rosactl"),
			},
			{
				Key:   stringPtr("red-hat-managed"),
				Value: stringPtr("true"),
			},
		},
		WaitTimeout: 15 * time.Minute,
	}

	fmt.Printf("Creating CloudFormation stack: %s\n", stackName)

	// Create stack
	output, err := cfnClient.CreateStack(ctx, createParams)
	if err != nil {
		// Check if stack already exists, try update instead
		if isAlreadyExistsError(err) {
			fmt.Printf("Stack already exists, attempting update...\n")
			return updateClusterVPC(ctx, cfnClient, event, stackName, templateBody)
		}
		return Response{Error: fmt.Sprintf("failed to create stack: %v", err)}, err
	}

	fmt.Printf("Stack created successfully: %s\n", output.StackID)

	return Response{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

// updateClusterVPC updates an existing cluster VPC CloudFormation stack
func updateClusterVPC(ctx context.Context, cfnClient *cloudformation.Client, event Event, stackName, templateBody string) (Response, error) {
	params := map[string]string{
		"ClusterName":      event.ClusterName,
		"SingleNatGateway": fmt.Sprintf("%t", event.SingleNatGateway),
	}

	// Add optional parameters
	if event.VpcCidr != "" {
		params["VpcCidr"] = event.VpcCidr
	}
	if len(event.PublicSubnetCidrs) > 0 {
		params["PublicSubnetCidrs"] = strings.Join(event.PublicSubnetCidrs, ",")
	}
	if len(event.PrivateSubnetCidrs) > 0 {
		params["PrivateSubnetCidrs"] = strings.Join(event.PrivateSubnetCidrs, ",")
	}
	if len(event.AvailabilityZones) >= 1 {
		params["AvailabilityZone1"] = event.AvailabilityZones[0]
	}
	if len(event.AvailabilityZones) >= 2 {
		params["AvailabilityZone2"] = event.AvailabilityZones[1]
	}
	if len(event.AvailabilityZones) >= 3 {
		params["AvailabilityZone3"] = event.AvailabilityZones[2]
	}

	updateParams := &cloudformation.UpdateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters:   params,
		WaitTimeout:  15 * time.Minute,
	}

	output, err := cfnClient.UpdateStack(ctx, updateParams)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to update stack: %v", err)}, err
	}

	fmt.Printf("Stack updated successfully: %s\n", output.StackID)

	return Response{
		StackID: output.StackID,
		Outputs: output.Outputs,
	}, nil
}

// deleteClusterVPC deletes the cluster VPC CloudFormation stack
func deleteClusterVPC(ctx context.Context, event Event) (Response, error) {
	if event.ClusterName == "" {
		return Response{Error: "cluster_name is required"}, fmt.Errorf("cluster_name is required")
	}

	fmt.Printf("Deleting cluster VPC CloudFormation stack for: %s\n", event.ClusterName)

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to load AWS config: %v", err)}, err
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Delete stack
	stackName := fmt.Sprintf("rosa-%s-vpc", event.ClusterName)
	err = cfnClient.DeleteStack(ctx, stackName, 15*time.Minute)
	if err != nil {
		return Response{Error: fmt.Sprintf("failed to delete stack: %v", err)}, err
	}

	fmt.Printf("Stack deleted successfully: %s\n", stackName)

	return Response{
		StackID: stackName,
		Outputs: map[string]string{"status": "deleted"},
	}, nil
}

// Start starts the Lambda handler
func Start() {
	lambda.Start(Handler)
}

// Helper functions

func readTemplateFile(filename string) (string, error) {
	// Try multiple possible locations
	locations := []string{
		filepath.Join("/app/templates", filename),        // Lambda container
		filepath.Join("templates", filename),              // Local development
		filepath.Join("../../templates", filename),        // Relative paths
		filepath.Join("../../../templates", filename),
	}

	// Also try relative to the executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		locations = append(locations, filepath.Join(exeDir, "templates", filename))
	}

	var lastErr error
	for _, location := range locations {
		data, err := os.ReadFile(location)
		if err == nil {
			return string(data), nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("template file %s not found in any of the expected locations: %w", filename, lastErr)
}

func stringPtr(s string) *string {
	return &s
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "already exists"
	return contains(err.Error(), "AlreadyExistsException") || contains(err.Error(), "already exists")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
