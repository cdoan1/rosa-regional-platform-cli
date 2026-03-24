package lambda

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/cloudformation/templates"
	"github.com/spf13/cobra"
)

const (
	defaultTimeout = 15 * time.Minute
)

type createOptions struct {
	imageURI          string
	functionName      string
	region            string
	stackName         string
	allowCrossAccount bool
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create FUNCTION_NAME",
		Short: "Create a Lambda function for cluster resource queries",
		Long: `Create a Lambda function that can describe cluster IAM and VPC resources.

This command deploys a Lambda function using a container image from ECR.
The Lambda function can be invoked to retrieve cluster resource information.

By default, only IAM principals in the same AWS account can invoke the Lambda.
Use --allow-cross-account to allow any AWS account with the Lambda ARN to invoke it.

Example:
  rosactl lambda create my-cluster-lambda \
    --image-uri 123456789012.dkr.ecr.us-east-1.amazonaws.com/rosactl:latest \
    --region us-east-1

  # Allow cross-account invocation
  rosactl lambda create my-cluster-lambda \
    --image-uri 123456789012.dkr.ecr.us-east-1.amazonaws.com/rosactl:latest \
    --region us-east-1 \
    --allow-cross-account`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.functionName = args[0]
			opts.stackName = fmt.Sprintf("rosa-lambda-%s", opts.functionName)
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.imageURI, "image-uri", "", "Container image URI from ECR (required)")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.Flags().BoolVar(&opts.allowCrossAccount, "allow-cross-account", false, "Allow any AWS account to invoke this Lambda (default: same account only)")

	_ = cmd.MarkFlagRequired("image-uri")
	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	fmt.Println("Creating Lambda function for cluster resource queries...")
	fmt.Printf("   Function name: %s\n", opts.functionName)
	fmt.Printf("   Stack name: %s\n", opts.stackName)
	fmt.Printf("   Container image: %s\n", opts.imageURI)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Println()

	// Read template
	templateBody, err := templates.Read("lambda-bootstrap.yaml")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewClient(cfg)

	// Prepare stack parameters
	crossAccountStr := "false"
	if opts.allowCrossAccount {
		crossAccountStr = "true"
	}

	params := &cloudformation.CreateStackParams{
		StackName:    opts.stackName,
		TemplateBody: templateBody,
		Parameters: map[string]string{
			"ContainerImageURI":       opts.imageURI,
			"FunctionName":            opts.functionName,
			"AllowCrossAccountInvoke": crossAccountStr,
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		Tags: []types.Tag{
			{
				Key:   stringPtr("ManagedBy"),
				Value: stringPtr("rosactl"),
			},
			{
				Key:   stringPtr("Component"),
				Value: stringPtr("cluster-query-lambda"),
			},
		},
		WaitTimeout: defaultTimeout,
	}

	fmt.Println("Creating CloudFormation stack...")

	// Create stack
	output, err := cfnClient.CreateStack(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Println()
	fmt.Println("Lambda function created successfully!")
	fmt.Println()
	fmt.Println("Outputs:")
	for key, value := range output.Outputs {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Println()
	fmt.Printf("To invoke the Lambda function:\n")
	fmt.Printf("  rosactl lambda invoke %s --cluster-name <cluster> --region %s\n", opts.functionName, opts.region)

	return nil
}

func stringPtr(s string) *string {
	return &s
}
