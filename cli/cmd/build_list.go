package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	buildListLimit  int
	buildListOffset int
)

var buildListCmd = &cobra.Command{
	Use:   "list [slug]",
	Short: "List builds for a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		builds, err := c.ListBuilds(args[0], buildListLimit, buildListOffset)
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(builds)
		}
		rows := make([][]string, len(builds))
		for i, b := range builds {
			rows[i] = []string{fmt.Sprint(b.ID), orDash(b.Ref), b.Status, b.CreatedAt}
		}
		ui.PrintTable([]string{"ID", "Ref", "Status", "Created"}, rows)
		return nil
	},
}

func init() {
	buildCmd.AddCommand(buildListCmd)
	buildListCmd.Flags().IntVar(&buildListLimit, "limit", 20, "max number of builds")
	buildListCmd.Flags().IntVar(&buildListOffset, "offset", 0, "number of builds to skip")
}
