package handler

import (
	"github.com/openshift-online/rosa-regional-platform-cli/internal/lambda"
	"github.com/spf13/cobra"
)

func NewHandlerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "handler",
		Short:  "Start the Lambda handler runtime",
		Long:   "Start the Lambda handler runtime. This command is used when rosactl runs as a Lambda function.",
		Hidden: true, // Hidden from help since it's only used by Lambda runtime
		Run: func(cmd *cobra.Command, args []string) {
			lambda.Start()
		},
	}

	return cmd
}
