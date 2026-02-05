package cmd

import (
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var versionSetLatestCmd = &cobra.Command{
	Use:   "set-latest [slug] [version]",
	Short: "Set a version as latest",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		t := true
		v, err := c.UpdateVersion(args[0], args[1], client.VersionUpdate{IsLatest: &t})
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(v)
		}
		ui.Success("Version " + args[1] + " is now latest.")
		return nil
	},
}

func init() {
	versionCmd.AddCommand(versionSetLatestCmd)
}
