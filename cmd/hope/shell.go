package cmd

import (
	"errors"
	"strings"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var shellCmdLabelsString string

func initShellCmd() {
	shellCmd.Flags().StringVarP(&shellCmdLabelsString, "selector", "l", "", "Exec in any pod matching the given selector")
}

var shellCmd = &cobra.Command{
	Use:   "shell [exec args]...",
	Short: "Start a shell, or run a command in the provided pod or any pod matching a label.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If the label argument is given, assume all arguments make up the
		//   command.
		// If the label argument isn't given, assume the first argument is the
		//   pod name, and the remainder make up the command.
		// If no command arguments are given, start an interactive shell.
		if len(args) == 0 && shellCmdLabelsString == "" {
			return errors.New("nothing to run against")
		}

		kubectl, err := utils.KubectlFromAnyMaster()
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		var podName string
		var commandArgs []string
		if shellCmdLabelsString != "" {
			output, err := kubeutil.GetKubectl(kubectl, "get", "pods", "-l", shellCmdLabelsString, "-o", "template={{range .items}}{{.metadata.name}} {{end}}")
			if err != nil {
				return err
			}

			pods := strings.Split(strings.TrimSpace(output), " ")
			podName = pods[0]
			commandArgs = args
		} else {
			podName = args[0]
			commandArgs = args[1:]
		}

		if len(commandArgs) == 0 {
			// Check to see if the pod will start a bash shell.
			if err := kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash", "-c", "exit"); err == nil {
				return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "bash")
			}

			return kubeutil.ExecKubectl(kubectl, "exec", "-it", podName, "--", "sh")
		} else {
			allArgs := []string{
				"exec",
				"-it",
				podName,
				"--",
			}
			allArgs = append(allArgs, commandArgs...)
			return kubeutil.ExecKubectl(kubectl, allArgs...)
		}
	},
}
