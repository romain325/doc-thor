package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var integrationGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get VCS integration details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		integration, err := c.GetVCSIntegration(name)
		if err != nil {
			return err
		}

		if cmd.Flag("json").Value.String() == "true" {
			data, _ := json.MarshalIndent(integration, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Styled output
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

		fmt.Println(headerStyle.Render(integration.Name))
		fmt.Println()

		enabledBadge := "✓ enabled"
		if !integration.Enabled {
			enabledBadge = "✗ disabled"
		}

		fmt.Printf("%s %s\n", labelStyle.Render("Provider:"), valueStyle.Render(integration.Provider))
		fmt.Printf("%s %s\n", labelStyle.Render("Instance URL:"), valueStyle.Render(integration.InstanceURL))
		fmt.Printf("%s %s\n", labelStyle.Render("Status:"), enabledBadge)
		fmt.Printf("%s %s\n", labelStyle.Render("Created:"), valueStyle.Render(integration.CreatedAt))
		fmt.Printf("%s %s\n", labelStyle.Render("Updated:"), valueStyle.Render(integration.UpdatedAt))

		return nil
	},
}

func init() {
	integrationCmd.AddCommand(integrationGetCmd)
}
