package lambda

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

type invokeOptions struct {
	functionName string
	clusterName  string
	region       string
	outputJSON   bool
}

func newInvokeCommand() *cobra.Command {
	opts := &invokeOptions{}

	cmd := &cobra.Command{
		Use:   "invoke FUNCTION_NAME",
		Short: "Invoke Lambda function to describe cluster resources",
		Long: `Invoke the Lambda function to retrieve cluster IAM and VPC resource information.

The Lambda function will query both cluster-iam and cluster-vpc CloudFormation
stacks and return their outputs in JSON format.

Example:
  rosactl lambda invoke my-cluster-lambda \
    --cluster-name test-cluster \
    --region us-east-2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.functionName = args[0]
			return runInvoke(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.clusterName, "cluster-name", "", "Name of the cluster to describe (required)")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region (required)")
	cmd.Flags().BoolVar(&opts.outputJSON, "output", false, "Output raw JSON response")

	cmd.MarkFlagRequired("cluster-name")
	cmd.MarkFlagRequired("region")

	return cmd
}

func runInvoke(ctx context.Context, opts *invokeOptions) error {
	fmt.Printf("Invoking Lambda function: %s\n", opts.functionName)
	fmt.Printf("   Cluster: %s\n", opts.clusterName)
	fmt.Printf("   Region: %s\n", opts.region)
	fmt.Println()

	// Create Lambda client
	client, err := lambda.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Prepare payload
	payload := map[string]interface{}{
		"action":       "describe-cluster",
		"cluster_name": opts.clusterName,
		"region":       opts.region,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Invoke function
	response, err := client.InvokeFunctionWithPayload(ctx, opts.functionName, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in response
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return fmt.Errorf("Lambda error: %s", errMsg)
	}

	// Output results
	if opts.outputJSON {
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(prettyJSON))
	} else {
		displayClusterInfo(result)
	}

	return nil
}

func displayClusterInfo(result map[string]interface{}) {
	clusterInfo, ok := result["cluster_info"].(map[string]interface{})
	if !ok {
		fmt.Println("No cluster information found in response")
		return
	}

	clusterName, _ := clusterInfo["cluster_name"].(string)
	region, _ := clusterInfo["region"].(string)

	fmt.Printf("Cluster: %s\n", clusterName)
	fmt.Printf("Region: %s\n", region)
	fmt.Println()

	// Display IAM info
	if iamInfo, ok := clusterInfo["iam"].(map[string]interface{}); ok {
		fmt.Println("IAM Resources:")
		fmt.Println("───────────────────────────────────────────────────────────────")
		fmt.Printf("  Stack Name: %s\n", iamInfo["stack_name"])
		fmt.Printf("  Status: %s\n", iamInfo["status"])
		if creationTime, ok := iamInfo["creation_time"].(string); ok {
			fmt.Printf("  Created: %s\n", creationTime)
		}

		if outputs, ok := iamInfo["outputs"].(map[string]interface{}); ok {
			fmt.Println("\n  Outputs:")
			for key, value := range outputs {
				fmt.Printf("    %-35s: %v\n", key, value)
			}
		}
		fmt.Println()
	}

	// Display VPC info
	if vpcInfo, ok := clusterInfo["vpc"].(map[string]interface{}); ok {
		fmt.Println("VPC Resources:")
		fmt.Println("───────────────────────────────────────────────────────────────")
		fmt.Printf("  Stack Name: %s\n", vpcInfo["stack_name"])
		fmt.Printf("  Status: %s\n", vpcInfo["status"])
		if creationTime, ok := vpcInfo["creation_time"].(string); ok {
			fmt.Printf("  Created: %s\n", creationTime)
		}

		if outputs, ok := vpcInfo["outputs"].(map[string]interface{}); ok {
			fmt.Println("\n  Outputs:")
			for key, value := range outputs {
				fmt.Printf("    %-35s: %v\n", key, value)
			}
		}
	}
}
