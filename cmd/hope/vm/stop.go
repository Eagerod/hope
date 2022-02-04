package vm

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/esxi"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a VM on the specified host.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		hypervisor, err := utils.HypervisorForNodeNamed(vmName)
		if err != nil {
			return err
		}

		hypervisorNode, err := (*hypervisor).UnderlyingNode()
		if err != nil {
			return err
		}

		return esxi.PowerOffVmNamed(hypervisorNode.ConnectionString(), vmName)
	},
}
