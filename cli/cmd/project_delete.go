package cmd

import (
	"github.com/charmbracelet/huh"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var projectDeleteCmd = &cobra.Command{
	Use:   "delete [slug]",
	Short: "Delete a project and all its builds/versions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]

		if !ui.JSON {
			var ok bool
			if err := huh.NewConfirm().
				Title("Delete project " + slug + "?").
				Description("This removes the project, all builds, and all versions permanently.").
				Value(&ok).
				Run(); err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}

		if err := c.DeleteProject(slug); err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(map[string]string{"deleted": slug})
		}
		ui.Success("Project " + slug + " deleted.")
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectDeleteCmd)
}
