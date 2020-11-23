package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a VM on the specified host.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going to start a VM.")
		return nil
	},
}
