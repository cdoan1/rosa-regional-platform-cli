package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// GitCommit is set via -ldflags during build
	GitCommit = "unknown"
	// Version is the semantic version
	Version = "dev"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  "Display the version and git commit hash of the rosactl binary.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("rosactl version %s (commit: %s)\n", Version, GitCommit)
		},
	}
	return cmd
}
