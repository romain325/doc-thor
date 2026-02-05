package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/config"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	loginUsername string
	loginPassword string
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and save session token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ui.JSON {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Username").
						Value(&loginUsername),
					huh.NewInput().
						Title("Password").
						EchoMode(huh.EchoModePassword).
						Value(&loginPassword),
				),
			).Run(); err != nil {
				return err
			}
		} else {
			if loginUsername == "" || loginPassword == "" {
				return fmt.Errorf("--username and --password are required with --json")
			}
		}

		url := cfg.Server.URL
		if url == "" {
			url = "http://localhost:8000"
		}
		loginClient := client.New(url, "") // no token yet
		resp, err := loginClient.Login(client.LoginRequest{
			Username: loginUsername,
			Password: loginPassword,
		})
		if err != nil {
			return err
		}

		cfg.Server.APIKey = resp.Token
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		if ui.JSON {
			return ui.PrintJSON(resp)
		}
		ui.Success("Logged in. Token saved.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authLoginCmd.Flags().StringVar(&loginUsername, "username", "", "username")
	authLoginCmd.Flags().StringVar(&loginPassword, "password", "", "password")
}
