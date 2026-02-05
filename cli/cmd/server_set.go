package cmd

import (
	"github.com/romain325/doc-thor/cli/internal/config"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var serverSetCmd = &cobra.Command{
	Use:   "set [url]",
	Short: "Set the target server URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Server.URL = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(map[string]string{"url": args[0]})
		}
		ui.Success("Server URL set to " + args[0])
		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverSetCmd)
}
