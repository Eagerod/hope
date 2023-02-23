package node

import (
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var hostnameCmdForce bool

func initHostnameCmdFlags() {
	hostnameCmd.Flags().BoolVarP(&hostnameCmdForce, "force", "", false, "try setting the hostname even if it appears to already be set")
}

var hostnameCmd = &cobra.Command{
	Use:   "hostname",
	Short: "Set the hostname on a node",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		hostname := args[1]

		node, err := utils.GetNode(nodeName)
		if err != nil {
			return err
		}

		if !node.IsRoleValid() {
			return fmt.Errorf("node %s has invalid role %s", node.Name, node.Role)
		}

		log.Info("Setting hostname on node ", node.Name, "(", node.Host, ")", " to ", hostname)

		return hope.SetHostname(log.WithFields(log.Fields{}), &node, hostname, hostnameCmdForce)
	},
}
