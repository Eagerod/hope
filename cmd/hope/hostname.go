package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
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
		host := args[0]
		hostname := args[1]

		if !nodePresentInConfig(host) {
			if !hostnameCmdForce {
				return hostNotFoundError(host)
			}
		}

		log.Info("Setting hostname on node ", host, " to ", hostname)

		return hope.SetHostname(log.WithFields(log.Fields{}), host, hostname, hostnameCmdForce)
	},
}
