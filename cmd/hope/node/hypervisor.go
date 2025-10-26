package node

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
)

var hypervisorCmd = &cobra.Command{
	Use:   "hypervisor <node-name>",
	Short: "print the hypervisor of the given node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := utils.GetBareNode(nodeName)
		if err != nil {
			return err
		}

		fmt.Println(node.Hypervisor)
		return nil
	},
}
