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

var kubeconfigCmdMergeFlag bool

func initKubeconfigCmdFlags() {
	kubeconfigCmd.Flags().BoolVarP(&kubeconfigCmdMergeFlag, "merge", "", false, "merge the kubeconfigs if one is already present locally")
}

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Fetch the kubeconfig from a master node",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var masterIp string
		masters := viper.GetStringSlice("masters")
		if len(args) == 0 {
			masterIp = masters[0]
			log.Debug("No host given to kubeconfig command. Using first master from masters list.")
		} else {
			masterIp = args[0]

			if !sliceutil.StringInSlice(masterIp, masters) {
				return errors.New(fmt.Sprintf("Failed to find master %s in config", masterIp))
			}
		}

		log.Debug("Fetching admin kubeconfig file from ", masterIp)

		return hope.FetchKubeconfig(log.WithFields(log.Fields{}), masterIp, kubeconfigCmdMergeFlag)
	},
}
