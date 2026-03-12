package clustervpc

import "github.com/spf13/cobra"

func NewClusterVPCCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-vpc",
		Short: "Manage cluster VPC resources",
		Long: `Manage VPC networking resources for ROSA hosted clusters.

This command group provides operations to create, delete, and inspect
VPC infrastructure for hosted cluster worker nodes.

The resources are created via CloudFormation stacks. Lambda bootstrap is optional
and no longer required for basic operations.`,
	}

	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDescribeCommand())

	return cmd
}
