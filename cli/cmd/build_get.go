package cmd

import (
	"fmt"
	"strconv"

	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var buildGetCmd = &cobra.Command{
	Use:   "get [slug] [build-id]",
	Short: "Show a build (includes logs)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid build id: %s", args[1])
		}
		build, err := c.GetBuild(args[0], uint(id))
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(build)
		}
		ui.DetailCard("Build", buildPairs(build))
		if build.Logs != "" {
			fmt.Println(ui.LogsStyle.Render(build.Logs))
		}
		return nil
	},
}

func init() {
	buildCmd.AddCommand(buildGetCmd)
}
