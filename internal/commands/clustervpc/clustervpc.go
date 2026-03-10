package clustervpc

import "github.com/spf13/cobra"

const (
	defaultLambdaFunction = "rosa-regional-platform-lambda"
)

func NewClusterVPCCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-vpc",
		Short: "Manage cluster VPC resources",
		Long: `Manage VPC networking resources for ROSA hosted clusters.

This command group provides operations to create, delete, and inspect
VPC infrastructure for hosted cluster worker nodes.`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDescribeCommand())

	return cmd
}
