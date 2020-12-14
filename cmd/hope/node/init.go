package node

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
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a node within the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := getNode(nodeName)
		if err != nil {
			return err
		}

		log.Info("Bootstrapping a node...")

		podNetworkCidr := viper.GetString("pod_network_cidr")
		masters := viper.GetStringSlice("masters")

		if node.IsMasterAndNode() {
			log.Info("Node ", node.Host, " appears to be both master and node. Creating master and removing NoSchedule taint...")

			if err := hope.CreateClusterMaster(log.WithFields(log.Fields{}), node, podNetworkCidr); err != nil {
				return err
			}

			kubectl, err := kubeutil.NewKubectlFromAnyNode(masters)
			if err != nil {
				return err
			}

			defer kubectl.Destroy()

			if err := hope.TaintNodeByHost(kubectl, node, "node-role.kubernetes.io/master:NoSchedule-"); err != nil {
				return err
			}
		} else if node.IsMaster() {
			return hope.CreateClusterMaster(log.WithFields(log.Fields{}), node, podNetworkCidr)
		} else if node.IsNode() {
			// Have to send in a master ip for it to grab a join token.
			aMaster := masters[0]

			if err := hope.CreateClusterNode(log.WithFields(log.Fields{}), node, aMaster); err != nil {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Failed to find node %s in config", nodeName))
		}

		return nil
	},
}
