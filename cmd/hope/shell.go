package cmd

import (
	"errors"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var shellCmdLabelsString string

func initShellCmd() {
	shellCmd.Flags().StringVarP(&shellCmdLabelsString, "selector", "l", "", "Exec in any pod matching the given selector")
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a shell in the provided pod.",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && shellCmdLabelsString == "" {
			return errors.New("Nothing to run against")
		}

		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		var podName string
		if len(args) == 1 {
			podName = args[0]
		} else if shellCmdLabelsString != "" {
			output, err := kubeutil.GetKubectl(kubectl, "get", "pods", "-l", shellCmdLabelsString, "-o", "template={{range .items}}{{.metadata.name}} {{end}}")
			if err != nil {
				return err
			}

			pods := strings.Split(strings.TrimSpace(output), " ")
			podName = pods[0]
		} else {
			return errors.New("Inconsistency issue")
		}

		// Check to see if the pod will start a bash shell.
		if err := kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash", "-c", "exit"); err == nil {
			return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash")
		}

		return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "sh")
	},
}
