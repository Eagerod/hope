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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists VMs on the specified host.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("node %s is not a hypervisor; cannot list VMs on it", hypervisor.Name)
		}

		list, err := esxi.ListVms(hypervisor.ConnectionString())
		if err != nil {
			return err
		}

		for _, l := range *list {
			fmt.Println(l)
		}

		return err
	},
}
