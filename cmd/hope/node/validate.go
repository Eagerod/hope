package node

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var validateCmdTypeSlice *[]string

func initValidateCmd() {
	validateCmdTypeSlice = validateCmd.Flags().StringArrayP("type", "t", []string{}, "validate nodes of this type")
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Checks every node in the manifest, and ensures that it's running on the expected hypervisor",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(*validateCmdTypeSlice) == 0 {
			*validateCmdTypeSlice = []string{
				hope.NodeRoleLoadBalancer.String(),
				hope.NodeRoleMaster.String(),
				hope.NodeRoleMasterAndNode.String(),
				hope.NodeRoleNode.String(),
			}
		}

		nodes, err := utils.GetBareNodeTypes(*validateCmdTypeSlice)
		if err != nil {
			return err
		}

		nodesByHypervisor := map[string][]hope.Node{}
		for _, node := range nodes {
			c, ok := nodesByHypervisor[node.Hypervisor]
			if !ok {
				c = []hope.Node{node}
			} else {
				c = append(c, node)
			}
			nodesByHypervisor[node.Hypervisor] = c
		}

		for k, v := range nodesByHypervisor {
			h, err := utils.GetHypervisor(k)
			if err != nil {
				return err
			}

			if err := h.ValidateNodes(v); err != nil {
				return err
			}
		}

		return nil
	},
}
