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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		masterIp := args[0]

		if !sliceutil.StringInSlice(masterIp, viper.GetStringSlice("masters")) {
			return errors.New(fmt.Sprintf("Failed to find master %s in config", masterIp))
		}

		return hope.FetchKubeconfig(log.WithFields(log.Fields{}), masterIp, kubeconfigCmdMergeFlag)
	},
}
