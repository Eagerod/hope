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

var resetCmdForce bool

func initResetCmd() {
	resetCmd.Flags().BoolVarP(&resetCmdForce, "force", "", false, "run kubeadm reset even if the node isn't a part of the cluster")
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Attempts to gracefully run kubeadm reset on the specified host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeName := args[0]
		masters := viper.GetStringSlice("masters")

		node, err := getNode(nodeName)
		if err != nil {
			return err
		}

		if !node.IsRoleValid() {
			return errors.New(fmt.Sprintf("Host (%s) not found in list of masters or nodes.", node.Host))
		}

		// If force is set, failing to find a kubeconfig shouldn't stop the
		//   command from brute force reseting the node.
		kubectl, err := kubeutil.NewKubectlFromAnyNode(masters)
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
		return hope.KubeadmResetRemote(log.WithFields(log.Fields{}), kubectl, node, resetCmdForce)
	},
}
