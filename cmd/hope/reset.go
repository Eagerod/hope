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

var resetCmdForce bool

func initResetCmd() {
	resetCmd.Flags().BoolVarP(&resetCmdForce, "force", "", false, "run kubeadm reset even if the node wasn't found in the cluster")
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Attempts to gracefully run kubeadm reset on the specified host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		masters := viper.GetStringSlice("masters")

		isMaster := sliceutil.StringInSlice(host, masters)
		isNode := sliceutil.StringInSlice(host, viper.GetStringSlice("nodes"))

		if !isMaster && !isNode {
			return errors.New(fmt.Sprintf("Host (%s) not found in list of masters or nodes.", host))
		}

		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		// TODO: may need to add more validation, like that this isn't the
		//   only master and is being removed, unless force is provided.
		return hope.KubeadmResetRemote(log.WithFields(log.Fields{}), kubectl, host, resetCmdForce)
	},
}
