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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a node within the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]

		log.Info("Bootstrapping a node...")

		podNetworkCidr := viper.GetString("pod_network_cidr")
		masters := viper.GetStringSlice("masters")

		isMaster := sliceutil.StringInSlice(host, masters)
		isWorker := sliceutil.StringInSlice(host, viper.GetStringSlice("nodes"))

		if isMaster && isWorker {
			log.Info("Node ", host, " appears in both master and node configurations. Creating master and removing NoSchedule taint...")

			if err := hope.CreateClusterMaster(log.WithFields(log.Fields{}), host, podNetworkCidr); err != nil {
				return err
			}

			if err := hope.TaintNodeByHost(host, "key:NoSchedule-"); err != nil {
				return err
			}
		}
		if isMaster {
			return hope.CreateClusterMaster(log.WithFields(log.Fields{}), host, podNetworkCidr)
		} else if isWorker {
			// Have to send in a master ip for it to grab a join token.
			aMaster := masters[0]

			if err := hope.CreateClusterNode(log.WithFields(log.Fields{}), host, aMaster); err != nil {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Failed to find node %s in config", host))
		}

		return nil
	},
}
