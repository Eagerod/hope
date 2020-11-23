package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves a VM to the specified host.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going to move a VM.")
		return nil
	},
}
