package lambda

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

var versionsOutputFormat string

func newVersionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions <name>",
		Short: "List all versions of a Lambda function",
		Long:  "Display all published versions of an AWS Lambda function.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			functionName := args[0]

			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			versions, err := client.ListVersions(cmd.Context(), functionName)
			if err != nil {
				return err
			}

			if len(versions) == 0 {
				fmt.Println("No versions found")
				return nil
			}

			if versionsOutputFormat == "json" {
				data, err := json.MarshalIndent(versions, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)

			if _, err := fmt.Fprintln(w, "VERSION\tDESCRIPTION\tLAST MODIFIED"); err != nil {
				return err
			}
			for _, v := range versions {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", v.Version, v.Description, v.LastModified); err != nil {
					return err
				}
			}

			return w.Flush()
		},
	}

	cmd.Flags().StringVarP(&versionsOutputFormat, "output", "o", "table", "Output format: table or json")

	return cmd
}
