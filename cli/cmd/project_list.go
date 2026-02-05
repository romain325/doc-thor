package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := c.ListProjects()
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(projects)
		}
		rows := make([][]string, len(projects))
		for i, p := range projects {
			rows[i] = []string{fmt.Sprint(p.ID), p.Slug, p.Name, p.SourceURL, p.CreatedAt}
		}
		ui.PrintTable([]string{"ID", "Slug", "Name", "Source URL", "Created"}, rows)
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
