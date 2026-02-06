package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var integrationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VCS integrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		integrations, err := c.ListVCSIntegrations()
		if err != nil {
			return err
		}

		if len(integrations) == 0 {
			fmt.Println("No VCS integrations found.")
			return nil
		}

		if cmd.Flag("json").Value.String() == "true" {
			data, _ := json.MarshalIndent(integrations, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Styled output
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
		rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

		fmt.Println(headerStyle.Render("VCS INTEGRATIONS"))
		fmt.Println()

		for _, integration := range integrations {
			enabledBadge := "✓ enabled"
			if !integration.Enabled {
				enabledBadge = "✗ disabled"
			}

			fmt.Printf("%s  %s  %s  %s\n",
				lipgloss.NewStyle().Bold(true).Render(integration.Name),
				lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(integration.Provider),
				rowStyle.Render(integration.InstanceURL),
				enabledBadge,
			)
		}

		return nil
	},
}

func init() {
	integrationCmd.AddCommand(integrationListCmd)
}
