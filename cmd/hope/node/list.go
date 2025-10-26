package node

import (
	"os"
	"text/template"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var listCmdTypeSlice *[]string
var listCmdTemplate *string

func initListCmd() {
	listCmdTypeSlice = listCmd.Flags().StringArrayP("type", "t", []string{}, "list nodes of this type")
	listCmdTemplate = listCmd.Flags().StringP("template", "", "{{.Name}}\n", "Format the output using this go-template")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "lists the nodes present in the hope config file.",
	Args:  cobra.ExactArgs(0),
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

		nodeNames, err := utils.GetBareNodeTypes(*listCmdTypeSlice)
		if err != nil {
			return err
		}

		tmpl, err := template.New("list-command-template").Parse(*listCmdTemplate)
		if err != nil {
			return err
		}

		for _, node := range nodeNames {
			tmpl.Execute(os.Stdout, node)
		}

		return nil
	},
}
