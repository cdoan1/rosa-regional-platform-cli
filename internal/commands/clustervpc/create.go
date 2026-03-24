package clustervpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/cloudformation"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/cloudformation/templates"
	"github.com/spf13/cobra"
)

type createOptions struct {
	clusterName        string
	region             string
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

This command creates a CloudFormation stack containing VPC, subnets, NAT gateways,
routing, security groups, and Route53 private hosted zone.

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
	cmd.Flags().StringVar(&opts.vpcCidr, "vpc-cidr", "10.0.0.0/16", "CIDR block for the VPC")
	cmd.Flags().StringVar(&opts.publicSubnetCidrs, "public-subnet-cidrs", "10.0.101.0/24,10.0.102.0/24,10.0.103.0/24", "Comma-separated public subnet CIDRs")
	cmd.Flags().StringVar(&opts.privateSubnetCidrs, "private-subnet-cidrs", "10.0.0.0/19,10.0.32.0/19,10.0.64.0/19", "Comma-separated private subnet CIDRs")
	cmd.Flags().StringVar(&opts.availabilityZones, "availability-zones", "", "Comma-separated availability zones (optional, auto-detected if empty)")
	cmd.Flags().BoolVar(&opts.singleNatGateway, "single-nat-gateway", true, "Use single NAT gateway (true=cost savings, false=HA per-AZ)")

	cmd.MarkFlagRequired("region")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	fmt.Printf("🌐 Creating cluster VPC resources for: %s\n", opts.clusterName)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Printf("   VPC CIDR: %s\n", opts.vpcCidr)
	fmt.Printf("   Single NAT Gateway: %t\n", opts.singleNatGateway)
	fmt.Println()

	// Read CloudFormation template
	fmt.Println("📄 Loading CloudFormation template...")
	templateBody, err := templates.Read("cluster-vpc.yaml")
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

	// Parse CIDR lists
	publicSubnets := strings.Split(opts.publicSubnetCidrs, ",")
	privateSubnets := strings.Split(opts.privateSubnetCidrs, ",")

	// Prepare stack parameters
	stackName := fmt.Sprintf("rosa-%s-vpc", opts.clusterName)
	params := map[string]string{
		"ClusterName":        opts.clusterName,
		"VpcCidr":            opts.vpcCidr,
		"PublicSubnetCidrs":  strings.Join(publicSubnets, ","),
		"PrivateSubnetCidrs": strings.Join(privateSubnets, ","),
		"SingleNatGateway":   fmt.Sprintf("%t", opts.singleNatGateway),
	}

	// Parse and add availability zones if provided
	if opts.availabilityZones != "" {
		azs := strings.Split(opts.availabilityZones, ",")
		if len(azs) >= 1 {
			params["AvailabilityZone1"] = azs[0]
		}
		if len(azs) >= 2 {
			params["AvailabilityZone2"] = azs[1]
		}
		if len(azs) >= 3 {
			params["AvailabilityZone3"] = azs[2]
		}
	}

	createParams := &cloudformation.CreateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters:   params,
		Tags: []types.Tag{
			{
				Key:   aws.String("Cluster"),
				Value: aws.String(opts.clusterName),
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

	fmt.Printf("☁️  Creating CloudFormation stack: %s\n", stackName)
	fmt.Println("   This may take several minutes...")
	fmt.Println()

	// Create stack
	output, err := cfnClient.CreateStack(ctx, createParams)
	if err != nil {
		// Check if stack already exists, try update instead
		var alreadyExistsErr *cloudformation.StackAlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			fmt.Println("ℹ️  Stack already exists, attempting update...")
			return updateVPCStack(ctx, cfnClient, stackName, templateBody, params)
		}

		// Get stack events to show what failed
		fmt.Println()
		fmt.Println("❌ Stack creation failed. Recent events:")
		events, evtErr := cfnClient.GetStackEvents(ctx, stackName, 10)
		if evtErr == nil && len(events) > 0 {
			for _, event := range events {
				if event.ResourceStatus == "CREATE_FAILED" || event.ResourceStatusReason != "" {
					fmt.Printf("   • %s: %s", event.LogicalResourceID, event.ResourceStatus)
					if event.ResourceStatusReason != "" {
						fmt.Printf(" - %s", event.ResourceStatusReason)
					}
					fmt.Println()
				}
			}
		}
		fmt.Println()

		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Println("✅ Cluster VPC resources created successfully!")
	fmt.Printf("   Stack ID: %s\n", output.StackID)
	fmt.Println()

	if len(output.Outputs) > 0 {
		fmt.Println("Outputs:")
		for key, value := range output.Outputs {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}

// updateVPCStack updates an existing cluster VPC CloudFormation stack
func updateVPCStack(ctx context.Context, cfnClient *cloudformation.Client,
	stackName, templateBody string, params map[string]string) error {
	updateParams := &cloudformation.UpdateStackParams{
		StackName:    stackName,
		TemplateBody: templateBody,
		Parameters:   params,
		WaitTimeout:  15 * time.Minute,
	}

	fmt.Println("   Updating stack...")
	output, err := cfnClient.UpdateStack(ctx, updateParams)
	if err != nil {
		// Check if no changes are needed
		var noChangesErr *cloudformation.NoChangesError
		if errors.As(err, &noChangesErr) {
			fmt.Println("ℹ️  No changes needed, stack is up to date")
			// Still get outputs
			stackOutput, err := cfnClient.GetStackOutputs(ctx, stackName)
			if err != nil {
				return fmt.Errorf("failed to get stack outputs: %w", err)
			}

			fmt.Println()
			fmt.Println("Stack Outputs:")
			for key, value := range stackOutput.Outputs {
				fmt.Printf("  %s: %s\n", key, value)
			}
			return nil
		}
		return fmt.Errorf("failed to update stack: %w", err)
	}

	fmt.Println("✅ Cluster VPC resources updated successfully!")
	fmt.Printf("   Stack ID: %s\n", output.StackID)
	fmt.Println()

	if len(output.Outputs) > 0 {
		fmt.Println("Updated Outputs:")
		for key, value := range output.Outputs {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	return nil
}
