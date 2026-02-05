package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	updateName        string
	updateSourceURL   string
	updateDockerImage string
)

var projectUpdateCmd = &cobra.Command{
	Use:   "update [slug]",
	Short: "Update a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.ProjectUpdate{}
		changed := false

		if cmd.Flags().Changed("name") {
			req.Name = updateName
			changed = true
		}
		if cmd.Flags().Changed("source-url") {
			req.SourceURL = updateSourceURL
			changed = true
		}
		if cmd.Flags().Changed("docker-image") {
			req.DockerImage = updateDockerImage
			changed = true
		}

		if !changed {
			return fmt.Errorf("nothing to update — provide at least one flag")
		}

		project, err := c.UpdateProject(args[0], req)
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(project)
		}
		ui.Success("Project updated.")
		ui.DetailCard("Project", [][]string{
			{"ID", fmt.Sprint(project.ID)},
			{"Slug", project.Slug},
			{"Name", project.Name},
			{"Source URL", project.SourceURL},
			{"Docker Image", project.DockerImage},
		})
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectUpdateCmd)
	projectUpdateCmd.Flags().StringVar(&updateName, "name", "", "new name")
	projectUpdateCmd.Flags().StringVar(&updateSourceURL, "source-url", "", "new git URL")
	projectUpdateCmd.Flags().StringVar(&updateDockerImage, "docker-image", "", "new Docker image")
}
