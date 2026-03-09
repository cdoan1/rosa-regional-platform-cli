package commands

import (
	"fmt"
	"os"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/commands/lambda"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/commands/oidc"
	"github.com/openshift-online/rosa-regional-platform-cli/internal/commands/version"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "rosactl",
	Short: "CLI tool for managing AWS resources",
	Long:  "rosactl is a command-line interface for managing AWS Lambda functions and other resources.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if region, _ := cmd.Flags().GetString("region"); region != "" {
			_ = os.Setenv("ROSACTL_REGION", region)
		}
		if profile, _ := cmd.Flags().GetString("profile"); profile != "" {
			_ = os.Setenv("ROSACTL_PROFILE", profile)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().String("region", "", "AWS region (overrides default)")
	rootCmd.PersistentFlags().String("profile", "", "AWS profile (overrides default)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.AddCommand(lambda.NewLambdaCommand())
	rootCmd.AddCommand(oidc.NewOIDCCommand())
	rootCmd.AddCommand(version.NewVersionCommand())
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func IsVerbose() bool {
	return verbose
}
