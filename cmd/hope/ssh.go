package cmd

import (
	"errors"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

var sshCmdForce bool

func initSshCmd() {
	sshCmd.Flags().BoolVarP(&sshCmdForce, "force", "", false, "set up SSH even if the node isn't a part of the cluster")
}

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Initializes SSH for the given host",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]

		// If a second argument is given, instead of trying to verify
		//   that the current host can SSH in without password, assume
		//   that it can, and just try to copy the SSH key provided by
		//   path to the remote.
		hasKeyArg := len(args) == 2

		if !nodePresentInConfig(host) {
			if !sshCmdForce {
				return errors.New(fmt.Sprintf("Host (%s) not found in list of masters or nodes.", host))
			}
		}

		if hasKeyArg {
			localKeyPath := args[1]
			return hope.CopySSHKeyToAuthorizedKeys(log.WithFields(log.Fields{}), localKeyPath, host)
		}

		return hope.EnsureSSHWithoutPassword(log.WithFields(log.Fields{}), host)
	},
}
