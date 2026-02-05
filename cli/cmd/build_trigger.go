package cmd

import (
	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	triggerRef string
	triggerTag string
)

var buildTriggerCmd = &cobra.Command{
	Use:   "trigger [slug]",
	Short: "Trigger a build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		build, err := c.TriggerBuild(args[0], client.BuildCreate{Ref: triggerRef, Tag: triggerTag})
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(build)
		}
		ui.Success("Build triggered.")
		ui.DetailCard("Build", buildPairs(build))
		return nil
	},
}

func init() {
	buildCmd.AddCommand(buildTriggerCmd)
	buildTriggerCmd.Flags().StringVar(&triggerRef, "ref", "", "git ref (branch, tag, or SHA)")
	buildTriggerCmd.Flags().StringVar(&triggerTag, "tag", "", "version tag for the published output")
}
