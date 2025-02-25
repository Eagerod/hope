package vm

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var createCmd = &cobra.Command{
	Use:   "create <image-name> <node-name>",
	Short: "Creates the named node as a VM using its defined hypervisor.",
	Long:  "Creates the named node as a VM using its defined hypervisor.\n\nArgs:\n  image-name: The name of the image from which to create a VM\n  node-name: The node to create from the image",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		nodeName := args[1]

		node, err := utils.GetBareNode(nodeName)
		if err != nil {
			return err
		}

		hypervisor, err := utils.GetHypervisor(node.Hypervisor)
		if err != nil {
			return err
		}

		vms, err := utils.GetVMs()
		if err != nil {
			return err
		}

		var vm hope.VMImageSpec
		for _, aVm := range vms.Images {
			if aVm.Name == vmName {
				vm = aVm
				break
			}
		}

		return hypervisor.CreateNode(node, vms, vm)
	},
}
