package vm

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/esxi"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("Node %s is not a hypervisor; cannot start a VM on it", hypervisor.Name)
		}

		return esxi.PowerOnVmNamed(hypervisor.ConnectionString(), vmName)
	},
}
