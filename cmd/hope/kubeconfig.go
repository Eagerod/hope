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

var kubeconfigCmdMergeFlag bool

func initKubeconfigCmdFlags() {
	kubeconfigCmd.Flags().BoolVarP(&kubeconfigCmdMergeFlag, "merge", "", false, "merge the kubeconfigs if one is already present locally")
}

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Fetch the kubeconfig from a master node",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {

		var master *hope.Node
		if len(args) == 0 {
			aMaster, err := getAnyMaster()
			if err != nil {
				return err
			}

			master = aMaster

			log.Debug("No host given to kubeconfig command. Using first master from nodes list.")
		} else {
			aMaster, err := getNode(args[0])
			if err != nil {
				return err
			}

			master = aMaster
		}

		if !master.IsMaster() {
			return errors.New(fmt.Sprintf("Node: %s is not a master node", master.Host))
		}

		log.Debug("Fetching admin kubeconfig file from ", master.Host)

		return hope.FetchKubeconfig(log.WithFields(log.Fields{}), master, kubeconfigCmdMergeFlag)
	},
}
