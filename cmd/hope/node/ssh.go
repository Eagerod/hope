package node

import (
	"net"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <node-name>",
	Short: "Initializes SSH for the given host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := utils.GetNode(nodeName)
		if err != nil {
			return err
		}

		if _, err := net.LookupHost(node.Host); err != nil {
			return err
		}

		return hope.EnsureSSHWithoutPassword(log.WithFields(log.Fields{}), &node)
	},
}
