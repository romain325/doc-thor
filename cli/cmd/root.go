package cmd

import (
	"os"

	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/config"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfg config.Config
	c   *client.Client
)

var rootCmd = &cobra.Command{
	Use:   "doc-thor",
	Short: "Automated technical documentation CLI",
	Long:  `doc-thor manages documentation projects, triggers builds, and publishes versioned docs.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}
		url := cfg.Server.URL
		if url == "" {
			url = "http://localhost:8000"
		}
		c = client.New(url, cfg.Server.APIKey)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&ui.JSON, "json", false, "output raw JSON")
}

// ---------------------------------------------------------------------------
// small helpers shared across cmd files
// ---------------------------------------------------------------------------

func boolStr(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
