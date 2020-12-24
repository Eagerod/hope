package cmd

import (
	"errors"
	"fmt"
	"regexp"
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
	Use:   "run",
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
				return errors.New(fmt.Sprintf("Parameter: %s not recognized", paramName))
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
		jobText, err := utils.ReplaceParametersInFile(job.File, fullArgsList)
		if err != nil {
			return err
		}

		kubectl, err := utils.KubectlFromAnyMaster()
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		output, err := hope.KubectlGetCreateStdIn(kubectl, jobText)
		if err != nil {
			return err
		}

		// Grab the job name from the output
		re, err := regexp.Compile("job\\.batch/([^\\s]+)")
		if err != nil {
			return err
		}

		kubeJobNameMatches := re.FindStringSubmatch(output)
		if len(kubeJobNameMatches) != 2 {
			return errors.New(fmt.Sprintf("Failed to parse job name from output: %s", output))
		}

		kubeJobName := kubeJobNameMatches[1]

		return hope.FollowLogsAndPollUntilJobComplete(log.WithFields(log.Fields{}), kubectl, kubeJobName, 10, 12)
	},
}
