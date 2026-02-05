package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		user, err := c.Me()
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(user)
		}
		ui.DetailCard("User", [][]string{
			{"ID", fmt.Sprint(user.ID)},
			{"Username", user.Username},
			{"Created", user.CreatedAt},
		})
		return nil
	},
}

func init() {
	authCmd.AddCommand(authWhoamiCmd)
}
