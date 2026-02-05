package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Manage builds",
}

func init() {
	rootCmd.AddCommand(buildCmd)
}

// buildPairs returns key-value pairs for a Build detail card.
func buildPairs(b client.Build) [][]string {
	pairs := [][]string{
		{"ID", fmt.Sprint(b.ID)},
		{"Project ID", fmt.Sprint(b.ProjectID)},
		{"Ref", orDash(b.Ref)},
		{"Status", ui.StatusBadge(b.Status)},
	}
	if b.StartedAt != "" {
		pairs = append(pairs, []string{"Started", b.StartedAt})
	}
	if b.FinishedAt != "" {
		pairs = append(pairs, []string{"Finished", b.FinishedAt})
	}
	return pairs
}
