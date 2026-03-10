package main

import (
	"fmt"
	"os"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/commands"
	lambdaHandler "github.com/openshift-online/rosa-regional-platform-cli/internal/lambda"
)

func main() {
	// Check if running in AWS Lambda environment
	// AWS Lambda sets the AWS_LAMBDA_RUNTIME_API environment variable
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		fmt.Println("Running in Lambda mode...")
		lambdaHandler.Start()
		return
	}

	// Otherwise, run as CLI
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
