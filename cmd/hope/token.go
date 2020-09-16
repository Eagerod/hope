package cmd

import (
	"encoding/base64"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var tokenCmdNamespace string

func initTokenCmd() {
	tokenCmd.Flags().StringVarP(&tokenCmdNamespace, "namespace", "n", "kube-system", "namespace in which to fetch the token")
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Fetch a service account token from the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		masters := viper.GetStringSlice("masters")

		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		// TODO: Should not live in cmd, but should be pushed into pkg/hope
		//   with some creative name.
		secretName, err := kubeutil.GetKubectl(kubectl, "-n", tokenCmdNamespace, "get", "serviceaccount", username, "-o", "jsonpath={.secrets[0].name}")
		if err != nil {
			return err
		}

		secretB64, err := kubeutil.GetKubectl(kubectl, "-n", tokenCmdNamespace, "get", "secret", secretName, "-o", "jsonpath={.data.token}")
		if err != nil {
			return err
		}

		secret, err := base64.StdEncoding.DecodeString(secretB64)
		if err != nil {
			return err
		}

		fmt.Println(string(secret))

		return nil
	},
}
