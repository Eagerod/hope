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

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Get the IP address of a VM on the specified hypervisor.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("Node %s is not a hypervisor; cannot find a node's IP from it", hypervisor.Name)
		}

		ip, err := esxi.GetIpAddressOfVmNamed(hypervisor.ConnectionString(), vmName)
		if err != nil {
			return err
		}

		if ip == "0.0.0.0" {
			return fmt.Errorf("VM %s has started, but has yet to be reachable", vmName)
		}

		fmt.Println(ip)

		return nil
	},
}
