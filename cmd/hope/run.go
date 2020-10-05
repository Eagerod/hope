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
	"github.com/spf13/viper"
)

import (
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

		job, err := getJob(jobName)
		if err != nil {
			return err
		}

		remainingParams := map[string]bool{}
		for _, param := range job.Parameters {
			remainingParams[param] = true
		}

		params := map[string]string{}
		for _, param := range *runCmdParameterSlice {
			components := strings.Split(param, "=")
			if len(components) != 2 {
				return errors.New(fmt.Sprintf("Failed to parse argument: %s", param))
			}

			paramName := components[0]
			paramValue := components[1]

			params[paramName] = paramValue
			if _, ok := remainingParams[paramName]; !ok {
				return errors.New(fmt.Sprintf("Parameter: %s not recognized", paramName))
			}

			remainingParams[paramName] = false
		}

		for param, missed := range remainingParams {
			if missed {
				return errors.New(fmt.Sprintf("Failed to find parameter: %s", param))
			}
		}

		// Pull kubeconfig from remote as late as possible to avoid extra
		//   network time before validation is done.
		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		// TODO: Move to pkg
		t, err := hope.TextSubstitutorFromFilepath(job.File)
		if err != nil {
			return err
		}

		if err := t.SubstituteTextFromMap(params); err != nil {
			return err
		}

		jobText := string(*t.Bytes)
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
