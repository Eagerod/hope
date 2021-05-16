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

var resetCmdForce bool
var resetCmdDeleteLocalData bool

func initResetCmd() {
	resetCmd.Flags().BoolVarP(&resetCmdForce, "force", "f", false, "run kubeadm reset even if the node isn't a part of the cluster")
	resetCmd.Flags().BoolVarP(&resetCmdDeleteLocalData, "delete-local-data", "d", false, "pass the --delete-local-data flag to kubectl drain")
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Attempts to gracefully run kubeadm reset on the specified host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]

		node, err := utils.GetNode(nodeName)
		if err != nil {
			return err
		}

		if !node.IsKubernetesNode() {
			return fmt.Errorf("Host (%s) not found in list of Kubernetes nodes", node.Host)
		}

		kubectl, err := utils.KubectlFromAnyMaster()
		if err != nil {
			if !resetCmdForce {
				return err
			}
		}

		// Since failing can still continue in forced cases, have to guard
		///   this.
		defer func() {
			if kubectl != nil {
				kubectl.Destroy()
			}
		}()

		// TODO: may need to add more validation, like that this isn't the
		//   only master and is being removed, unless force is provided.
		return hope.KubeadmResetRemote(log.WithFields(log.Fields{}), kubectl, &node, resetCmdDeleteLocalData, resetCmdForce)
	},
}
