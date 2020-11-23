package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Creates a VM image from the defined packer spec.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("I'm going to create a VM image.")
		return nil
	},
}
