package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var integrationDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a VCS integration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Confirmation prompt
		var confirm bool
		err := huh.NewConfirm().
			Title(fmt.Sprintf("Delete VCS integration '%s'?", name)).
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()
		if err != nil {
			return err
		}

		if !confirm {
			fmt.Println("Cancelled.")
			return nil
		}

		if err := c.DeleteVCSIntegration(name); err != nil {
			return err
		}

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Println(successStyle.Render("✓ VCS integration deleted"))

		return nil
	},
}

func init() {
	integrationCmd.AddCommand(integrationDeleteCmd)
}
