package node

import (
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var initCmdForce bool

func initInitCmd() {
	initCmd.Flags().BoolVarP(&initCmdForce, "force", "f", false, "skip hostname verification")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a node within the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := utils.GetNode(nodeName)
		if err != nil {
			return err
		}

		// Load balancer have a super lightweight init, so run its init before
		//   fetching some potentially heavier state from the cluster.
		if node.IsLoadBalancer() {
			return hope.InitLoadBalancer(log.WithFields(log.Fields{}), &node)
		}

		podNetworkCidr := viper.GetString("pod_network_cidr")
		masters, err := utils.GetAvailableMasters()
		if err != nil {
			return err
		}

		loadBalancer, err := utils.GetLoadBalancer()
		if err != nil && loadBalancer != (hope.Node{}) {
			return err
		}
		loadBalancerHost := viper.GetString("load_balancer_host")

		if node.IsMasterAndNode() {
			log.Info("Node ", node.Host, " appears to be both master and node. Creating master and removing NoSchedule taint...")

			if err := hope.CreateClusterMaster(log.WithFields(log.Fields{}), &node, podNetworkCidr, &loadBalancer, loadBalancerHost, &masters, initCmdForce); err != nil {
				return err
			}

			kubectl, err := utils.KubectlFromAnyMaster()
			if err != nil {
				return err
			}

			defer kubectl.Destroy()

			return hope.TaintNodeByHost(kubectl, &node, "node-role.kubernetes.io/master:NoSchedule-")
		} else if node.IsMaster() {
			return hope.CreateClusterMaster(log.WithFields(log.Fields{}), &node, podNetworkCidr, &loadBalancer, loadBalancerHost, &masters, initCmdForce)
		} else if node.IsNode() {
			return hope.CreateClusterNode(log.WithFields(log.Fields{}), &node, &masters, initCmdForce)
		} else {
			return fmt.Errorf("failed to find node %s in config", nodeName)
		}
	},
}
