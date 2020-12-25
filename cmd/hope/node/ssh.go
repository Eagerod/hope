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
	Use:   "ssh",
	Short: "Initializes SSH for the given host",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := utils.GetNode(nodeName)
		if err != nil {
			return err
		}

		// If a second argument is given, instead of trying to verify
		//   that the current host can SSH in without password, assume
		//   that it can, and just try to copy the SSH key provided by
		//   path to the remote.
		hasKeyArg := len(args) == 2

		if hasKeyArg {
			localKeyPath := args[1]
			return hope.CopySSHKeyToAuthorizedKeys(log.WithFields(log.Fields{}), localKeyPath, node)
		}


		if _, err := net.LookupHost(node.Host); err != nil {
			return err
		}

		return hope.EnsureSSHWithoutPassword(log.WithFields(log.Fields{}), node)
	},
}
