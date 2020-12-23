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
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var initCmdForce bool

func initInitCmd() {
	initCmd.Flags().BoolVarP(&initCmdForce, "force", "f", false, "don't ask the user to verify the hostname")
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

		log.Info("Bootstrapping a node...")

		podNetworkCidr := viper.GetString("pod_network_cidr")
		masters, err := utils.GetMasters()
		if err != nil {
			return err
		}

		if node.IsMasterAndNode() {
			log.Info("Node ", node.Host, " appears to be both master and node. Creating master and removing NoSchedule taint...")

			if err := hope.CreateClusterMaster(log.WithFields(log.Fields{}), node, podNetworkCidr, initCmdForce); err != nil {
				return err
			}

			masterIps := []string{}
			for _, master := range *masters {
				masterIps = append(masterIps, master.ConnectionString())
			}

			kubectl, err := kubeutil.NewKubectlFromAnyNode(masterIps)
			if err != nil {
				return err
			}

			defer kubectl.Destroy()

			if err := hope.TaintNodeByHost(kubectl, node, "node-role.kubernetes.io/master:NoSchedule-"); err != nil {
				return err
			}
		} else if node.IsMaster() {
			return hope.CreateClusterMaster(log.WithFields(log.Fields{}), node, podNetworkCidr, initCmdForce)
		} else if node.IsNode() {
			if err := hope.CreateClusterNode(log.WithFields(log.Fields{}), node, masters, initCmdForce); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Failed to find node %s in config", nodeName)
		}

		return nil
	},
}
