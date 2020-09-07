package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configs and secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going set a configuration value or file, and maybe secrets...")
		return nil
	},
}
