package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/spf13/cobra"
)

var (
	createIntName          string
	createIntProvider      string
	createIntURL           string
	createIntToken         string
	createIntWebhookSecret string
	createIntEnabled       bool
)

var integrationCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new VCS integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Debug: print what we received
		fmt.Fprintf(os.Stderr, "DEBUG: name=%q provider=%q url=%q token=%q secret=%q enabled=%v\n",
			createIntName, createIntProvider, createIntURL, createIntToken, createIntWebhookSecret, createIntEnabled)

		req := client.VCSIntegrationCreate{
			Name:          createIntName,
			Provider:      createIntProvider,
			InstanceURL:   createIntURL,
			AccessToken:   createIntToken,
			WebhookSecret: createIntWebhookSecret,
			Enabled:       createIntEnabled,
		}

		integration, err := c.CreateVCSIntegration(req)
		if err != nil {
			return err
		}

		if cmd.Flag("json").Value.String() == "true" {
			data, _ := json.MarshalIndent(integration, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		fmt.Println(successStyle.Render("✓ VCS integration created"))
		fmt.Printf("  Name: %s\n", integration.Name)
		fmt.Printf("  Provider: %s\n", integration.Provider)
		fmt.Printf("  URL: %s\n", integration.InstanceURL)

		return nil
	},
}

func init() {
	integrationCreateCmd.Flags().StringVar(&createIntName, "name", "", "Unique integration name (required)")
	integrationCreateCmd.Flags().StringVar(&createIntProvider, "provider", "", "Provider type: gitlab, github, gitea (required)")
	integrationCreateCmd.Flags().StringVar(&createIntURL, "url", "", "VCS instance URL (required)")
	integrationCreateCmd.Flags().StringVar(&createIntToken, "token", "", "API access token (required)")
	integrationCreateCmd.Flags().StringVar(&createIntWebhookSecret, "webhook-secret", "", "Webhook secret for signature validation (required)")
	integrationCreateCmd.Flags().BoolVar(&createIntEnabled, "enabled", true, "Enable integration")

	integrationCreateCmd.MarkFlagRequired("name")
	integrationCreateCmd.MarkFlagRequired("provider")
	integrationCreateCmd.MarkFlagRequired("url")
	integrationCreateCmd.MarkFlagRequired("token")
	integrationCreateCmd.MarkFlagRequired("webhook-secret")

	integrationCmd.AddCommand(integrationCreateCmd)
}
