package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var projectGetCmd = &cobra.Command{
	Use:   "get [slug]",
	Short: "Show a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := c.GetProject(args[0])
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(project)
		}
		ui.DetailCard("Project", [][]string{
			{"ID", fmt.Sprint(project.ID)},
			{"Slug", project.Slug},
			{"Name", project.Name},
			{"Source URL", project.SourceURL},
			{"Docker Image", project.DockerImage},
			{"Created", project.CreatedAt},
			{"Updated", project.UpdatedAt},
		})
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectGetCmd)
}
