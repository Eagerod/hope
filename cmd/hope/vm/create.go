package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a VM on the specified host.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going to create a VM.")
		return nil
	},
}
