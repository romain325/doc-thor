package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	createSlug        string
	createName        string
	createSourceURL   string
	createDockerImage string
)

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.ProjectCreate{
			Slug:        createSlug,
			Name:        createName,
			SourceURL:   createSourceURL,
			DockerImage: createDockerImage,
		}

		project, err := c.CreateProject(req)
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(project)
		}
		ui.Success("Project created.")
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
	projectCmd.AddCommand(projectCreateCmd)
	projectCreateCmd.Flags().StringVar(&createSlug, "slug", "", "unique project identifier")
	projectCreateCmd.Flags().StringVar(&createName, "name", "", "human-readable name")
	projectCreateCmd.Flags().StringVar(&createSourceURL, "source-url", "", "git repository URL")
	projectCreateCmd.Flags().StringVar(&createDockerImage, "docker-image", "", "Docker image the builder will run for this project")
	_ = projectCreateCmd.MarkFlagRequired("slug")
	_ = projectCreateCmd.MarkFlagRequired("name")
	_ = projectCreateCmd.MarkFlagRequired("source-url")
	_ = projectCreateCmd.MarkFlagRequired("docker-image")
}
