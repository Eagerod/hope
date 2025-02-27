package cmd

import (
	"fmt"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
)

var runCmdParameterSlice *[]string

func initRunCmdFlags() {
	runCmdParameterSlice = runCmd.Flags().StringArrayP("param", "p", []string{}, "parameters to populate in the job yaml")
}

var runCmd = &cobra.Command{
	Use:   "run <job-name>",
	Short: "Execute and follow a Kubernetes job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobName := args[0]

		job, err := utils.GetJob(jobName)
		if err != nil {
			return err
		}

		// Combine args given from the command line, and ones not given to let
		//   the parameter substitution fall back to env when available.
		// Would probably be faster to just populate from the slice, then from
		//   any remaining args via env, but this adds some extra validation
		//   that would otherwise go unchecked.
		fullArgsList := []string{}

		remainingParams := map[string]bool{}
		for _, param := range job.Parameters {
			remainingParams[param] = true
		}

		for _, param := range *runCmdParameterSlice {
			components := strings.SplitN(param, "=", 2)
			paramName := components[0]

			if _, ok := remainingParams[paramName]; !ok {
				return fmt.Errorf("parameter: %s not recognized", paramName)
			}

			remainingParams[paramName] = false
			fullArgsList = append(fullArgsList, param)
		}

		for param, missed := range remainingParams {
			if missed {
				fullArgsList = append(fullArgsList, param)
			}
		}

		// TODO: Move to pkg
		jobText, err := hope.ReplaceParametersInFile(job.File, fullArgsList)
		if err != nil {
			return err
		}

		kubectl, err := utils.KubectlFromAnyMaster()
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		output, err := hope.KubectlGetCreateStdIn(kubectl, jobText, "-o", "template={{.metadata.namespace}}/{{.metadata.name}}")
		if err != nil {
			return err
		}

		return hope.FollowLogsAndPollUntilJobComplete(log.WithFields(log.Fields{}), kubectl, output, 10, 12)
	},
}
