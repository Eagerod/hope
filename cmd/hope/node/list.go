package node

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var listCmdTypeSlice *[]string

func initListCmd() {
	listCmdTypeSlice = listCmd.Flags().StringArrayP("type", "t", []string{}, "list nodes of this type")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "lists the nodes present in the hope config file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(*listCmdTypeSlice) == 0 {
			*listCmdTypeSlice = []string{
				hope.NodeRoleHypervisor.String(),
				hope.NodeRoleLoadBalancer.String(),
				hope.NodeRoleMaster.String(),
				hope.NodeRoleMasterAndNode.String(),
				hope.NodeRoleNode.String(),
			}
		}

		nodeNames, err := utils.GetNodeNames(*listCmdTypeSlice)
		if err != nil {
			return err
		}

		for _, node := range nodeNames {
			fmt.Println(node)
		}

		return nil
	},
}
