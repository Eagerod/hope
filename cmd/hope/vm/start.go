package vm

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a VM on the specified host.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		hypervisor, err := utils.HypervisorForNodeNamed(vmName)
		if err != nil {
			return err
		}

		return (*hypervisor).StartVM(vmName)
	},
}
