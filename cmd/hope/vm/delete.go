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

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("Node %s is not a hypervisor; cannot delete a VM on it", hypervisor.Name)
		}

		connectionString := hypervisor.ConnectionString()

		// If the VM is on, don't allow the user to proceed, and force them to
		//   shut it off themselves.
		powerState, err := esxi.PowerStateOfVmNamed(connectionString, vmName)
		if err != nil {
			return err
		}

		if powerState != esxi.VmStatePoweredOff {
			return fmt.Errorf("VM %s has power state: %s; cannot delete", vmName, powerState)
		}

		return esxi.DeleteVmNamed(connectionString, vmName)
	},
}
