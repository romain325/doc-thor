package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var versionListCmd = &cobra.Command{
	Use:   "list [slug]",
	Short: "List versions for a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versions, err := c.ListVersions(args[0])
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(versions)
		}
		rows := make([][]string, len(versions))
		for i, v := range versions {
			rows[i] = []string{v.Version, fmt.Sprint(v.BuildID), boolStr(v.Published), boolStr(v.IsLatest), v.CreatedAt}
		}
		ui.PrintTable([]string{"Version", "Build ID", "Published", "Latest", "Created"}, rows)
		return nil
	},
}

func init() {
	versionCmd.AddCommand(versionListCmd)
}
