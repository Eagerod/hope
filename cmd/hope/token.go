package cmd

import (
	"encoding/base64"
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var tokenCmdNamespace string

func initTokenCmd() {
	tokenCmd.Flags().StringVarP(&tokenCmdNamespace, "namespace", "n", "kube-system", "namespace in which to fetch the token")
}

var tokenCmd = &cobra.Command{
	Use:   "token <service-account-name>",
	Short: "Fetch a service account token from the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		kubectl, err := utils.KubectlFromAnyMaster()
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
