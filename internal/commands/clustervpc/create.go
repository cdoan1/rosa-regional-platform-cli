package clustervpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

type createOptions struct {
	clusterName        string
	region             string
	lambdaFunction     string
	vpcCidr            string
	publicSubnetCidrs  string
	privateSubnetCidrs string
	availabilityZones  string
	singleNatGateway   bool
}

func newCreateCommand() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create CLUSTER_NAME",
		Short: "Create cluster VPC resources",
		Long: `Create VPC networking resources for a hosted cluster.

This command invokes the Lambda function to create a CloudFormation stack
containing VPC, subnets, NAT gateways, routing, security groups, and
Route53 private hosted zone.

Example:
  rosactl cluster-vpc create my-cluster --region us-east-1

  # With custom CIDR ranges
  rosactl cluster-vpc create my-cluster \
    --region us-east-1 \
    --vpc-cidr 10.1.0.0/16 \
    --public-subnet-cidrs 10.1.101.0/24,10.1.102.0/24,10.1.103.0/24 \
    --private-subnet-cidrs 10.1.0.0/19,10.1.32.0/19,10.1.64.0/19

  # With specific availability zones and per-AZ NAT gateways
  rosactl cluster-vpc create my-cluster \
    --region us-east-1 \
    --availability-zones us-east-1a,us-east-1b,us-east-1c \
    --single-nat-gateway=false`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.clusterName = args[0]
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.Flags().StringVar(&opts.lambdaFunction, "lambda-function", defaultLambdaFunction, "Name of the Lambda function")
	cmd.Flags().StringVar(&opts.vpcCidr, "vpc-cidr", "10.0.0.0/16", "CIDR block for the VPC")
	cmd.Flags().StringVar(&opts.publicSubnetCidrs, "public-subnet-cidrs", "10.0.101.0/24,10.0.102.0/24,10.0.103.0/24", "Comma-separated public subnet CIDRs")
	cmd.Flags().StringVar(&opts.privateSubnetCidrs, "private-subnet-cidrs", "10.0.0.0/19,10.0.32.0/19,10.0.64.0/19", "Comma-separated private subnet CIDRs")
	cmd.Flags().StringVar(&opts.availabilityZones, "availability-zones", "", "Comma-separated availability zones (optional, auto-detected if empty)")
	cmd.Flags().BoolVar(&opts.singleNatGateway, "single-nat-gateway", true, "Use single NAT gateway (true=cost savings, false=HA per-AZ)")

	cmd.MarkFlagRequired("region")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	fmt.Printf("Creating cluster VPC resources for: %s\n", opts.clusterName)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Printf("   VPC CIDR: %s\n", opts.vpcCidr)
	fmt.Printf("   Single NAT Gateway: %t\n", opts.singleNatGateway)
	fmt.Println()

	// Create Lambda client
	lambdaClient, err := lambda.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Parse CIDR lists
	publicSubnets := strings.Split(opts.publicSubnetCidrs, ",")
	privateSubnets := strings.Split(opts.privateSubnetCidrs, ",")

	// Parse availability zones if provided
	var azs []string
	if opts.availabilityZones != "" {
		azs = strings.Split(opts.availabilityZones, ",")
	}

	// Prepare Lambda payload
	payload := map[string]interface{}{
		"action":              "apply-cluster-vpc",
		"cluster_name":        opts.clusterName,
		"vpc_cidr":            opts.vpcCidr,
		"public_subnet_cidrs": publicSubnets,
		"private_subnet_cidrs": privateSubnets,
		"single_nat_gateway":  opts.singleNatGateway,
	}
	if len(azs) > 0 {
		payload["availability_zones"] = azs
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Printf("Invoking Lambda function: %s\n", opts.lambdaFunction)

	// Invoke Lambda
	result, err := lambdaClient.InvokeFunctionWithPayload(ctx, opts.lambdaFunction, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	// Parse response
	var response lambdaResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return fmt.Errorf("failed to parse Lambda response: %w", err)
	}

	// Check for errors
	if response.Error != "" {
		return fmt.Errorf("Lambda execution failed: %s", response.Error)
	}

	fmt.Println("Cluster VPC resources created successfully!")
	fmt.Println()
	fmt.Println("Stack ID:", response.StackID)
	fmt.Println()
	fmt.Println("Outputs:")
	for key, value := range response.Outputs {
		fmt.Printf("  %s: %s\n", key, value)
	}

	return nil
}

type lambdaResponse struct {
	StackID string            `json:"stack_id"`
	Outputs map[string]string `json:"outputs"`
	Error   string            `json:"error,omitempty"`
}
