package cmd

import (
	"errors"
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
			nodes, err := getNodes()
			if err != nil {
				return err
			}
	
			for _, node := range *nodes {
				if node.IsMaster() {
					master = &node
					break
				}
			}
			if master == nil {
				return errors.New("Failed to find any master in nodes config")
			}

			log.Debug("No host given to kubeconfig command. Using first master from nodes list.")
		} else {
			aMaster, err := getNode(args[0])
			if err != nil {
				return err
			}

			master = aMaster
		}

		log.Debug("Fetching admin kubeconfig file from ", master.Host)

		return hope.FetchKubeconfig(log.WithFields(log.Fields{}), master, kubeconfigCmdMergeFlag)
	},
}
