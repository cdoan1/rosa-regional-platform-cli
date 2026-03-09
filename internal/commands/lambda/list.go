package lambda

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/aws/lambda"
	"github.com/spf13/cobra"
)

var outputFormat string

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all Lambda functions",
		Long:  "Display all AWS Lambda functions with their names and creation dates.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := lambda.NewClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			functions, err := client.ListFunctions(cmd.Context())
			if err != nil {
				return err
			}

			if outputFormat == "json" {
				data, err := json.MarshalIndent(functions, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)

			if _, err := fmt.Fprintln(w, "NAME\tCREATED\tVERSION"); err != nil {
				return err
			}
			for _, fn := range functions {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", fn.Name, fn.Created, fn.LatestVersion); err != nil {
					return err
				}
			}

			return w.Flush()
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")

	return cmd
}
