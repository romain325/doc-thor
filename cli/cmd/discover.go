package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [integration-name] [scope]",
	Short: "Discover projects with .doc-thor.project.yaml in VCS scope",
	Long: `Scan a VCS scope (group, organization, namespace) for projects with .doc-thor.project.yaml.

Examples:
  doc-thor discover company-gitlab myteam/docs
  doc-thor discover github-org mycompany`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		integrationName := args[0]
		scope := args[1]

		req := client.DiscoveryRequest{
			Scope: scope,
		}

		result, err := c.DiscoverProjects(integrationName, req)
		if err != nil {
			return err
		}

		if cmd.Flag("json").Value.String() == "true" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Styled output
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

		fmt.Println(headerStyle.Render("PROJECT DISCOVERY"))
		fmt.Println()

		if result.Count == 0 {
			fmt.Println("No projects with .doc-thor.project.yaml found in scope:", scope)
			return nil
		}

		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Found %d projects", result.Count)))
		fmt.Println()

		for i, proj := range result.Projects {
			fmt.Printf("%s %s\n",
				lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d.", i+1)),
				lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(proj.Path),
			)
			fmt.Printf("   %s %s\n", labelStyle.Render("Name:"), proj.Name)
			if proj.DocThorConfig != nil {
				fmt.Printf("   %s %s\n", labelStyle.Render("Slug:"), proj.DocThorConfig.Slug)
				fmt.Printf("   %s %s\n", labelStyle.Render("Image:"), proj.DocThorConfig.DockerImage)
				if len(proj.DocThorConfig.BranchMappings) > 0 {
					fmt.Printf("   %s %d mappings\n", labelStyle.Render("Branches:"), len(proj.DocThorConfig.BranchMappings))
				}
			}
			fmt.Println()
		}

		fmt.Println(lipgloss.NewStyle().Faint(true).Render(
			"Use 'doc-thor project import' to import these projects."))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)
}
