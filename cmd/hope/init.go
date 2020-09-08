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
	Short: "Bootstrap the master node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Bootstrapping a master node...")

		masterIp := args[0]
		podNetworkCidr := viper.GetString("pod_network_cidr")

		if !sliceutil.StringInSlice(masterIp, viper.GetStringSlice("masters")) {
			return errors.New(fmt.Sprintf("Failed to find master %s in config", masterIp))
		}

		return hope.CreateClusterMaster(log.WithFields(log.Fields{}), masterIp, podNetworkCidr)
	},
}
