package vm

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists VMs on the specified host.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]

		hypervisor, err := utils.GetHypervisor(hypervisorName)
		if err != nil {
			return err
		}

		list, err := hypervisor.ListNodes()
		if err != nil {
			return err
		}

		for _, l := range list {
			fmt.Println(l)
		}

		return err
	},
}
