package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a shell in the provided pod.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		// Check to see if the pod will start a bash shell.
		if err := kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash", "-c", "exit"); err == nil {
			return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash")
		}

		return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "sh")
	},
}
