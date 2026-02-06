package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var integrationTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test VCS integration connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		result, err := c.TestVCSIntegration(name)
		if err != nil {
			return err
		}

		if cmd.Flag("json").Value.String() == "true" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if result.Success {
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
			fmt.Println(successStyle.Render("✓ Connection successful"))
			if result.Message != "" {
				fmt.Printf("  %s\n", result.Message)
			}
		} else {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
			fmt.Println(errorStyle.Render("✗ Connection failed"))
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			}
		}

		return nil
	},
}

func init() {
	integrationCmd.AddCommand(integrationTestCmd)
}
