package cmd

import (
	"errors"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/sliceutil"
)

var sshCmdForce bool

func initSshCmd() {
	sshCmd.Flags().BoolVarP(&sshCmdForce, "force", "", false, "set up SSH even if the node isn't a part of the cluster")
}

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Initializes SSH for the given host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]

		isMaster := sliceutil.StringInSlice(host, viper.GetStringSlice("masters"))
		isNode := sliceutil.StringInSlice(host, viper.GetStringSlice("nodes"))

		if !isMaster && !isNode {
			if !sshCmdForce {
				return errors.New(fmt.Sprintf("Host (%s) not found in list of masters or nodes.", host))
			}
		}

		return hope.EnsureSSHWithoutPassword(log.WithFields(log.Fields{}), host)
	},
}
