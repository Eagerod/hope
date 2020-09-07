package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap the master node",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going to install a bunch of dependencies, and turn this into a control plane node...")
		return nil
	},
}
