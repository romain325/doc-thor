package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server health and backend status",
	RunE: func(cmd *cobra.Command, args []string) error {
		health, err := c.Health()
		if err != nil {
			return fmt.Errorf("server unreachable: %w", err)
		}

		backends, backendErr := c.Backends()

		if ui.JSON {
			out := map[string]any{"health": health}
			if backendErr == nil {
				out["backends"] = backends
			}
			return ui.PrintJSON(out)
		}

		ui.DetailCard("Server", [][]string{
			{"Status", ui.StatusBadge(health["status"])},
			{"URL", cfg.Server.URL},
		})

		if backendErr != nil {
			fmt.Println(ui.WarningStyle.Render("Backends: " + backendErr.Error()))
			return nil
		}

		rows := make([][]string, len(backends))
		for i, b := range backends {
			rows[i] = []string{b.Name, b.URL, boolStr(b.Healthy), b.LastCheck}
		}
		fmt.Println()
		ui.PrintTable([]string{"Backend", "URL", "Healthy", "Last Check"}, rows)
		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverStatusCmd)
}
