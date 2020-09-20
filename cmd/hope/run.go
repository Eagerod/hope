package cmd

import (
	"io/ioutil"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/envsubst"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
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

		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

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

		// TODO: Move to pkg
		jobContents, err := ioutil.ReadFile(job.File)
		if err != nil {
			return err
		}

		jobText, err := envsubst.GetEnvsubstArgs(params, string(jobContents))
		if err != nil {
			return err
		}

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
		kubeJobSelector := fmt.Sprintf("job-name=%s", kubeJobName)
		
		// Wait for this loop to finish without failure.
		// Exponential backoff for re-attempts, max 12 seconds
		attempts := 1
		for ok := false; !ok; {
			if err := kubeutil.ExecKubectl(kubectl, "logs", "-f", "-l", kubeJobSelector); err != nil {
				log.Warn(err)
				attemptsDuration := math.Pow(2, float64(attempts-1))
				sleepSeconds := int(math.Min(attemptsDuration, 12))

				if sleepSeconds == 12 {
					log.Debug("Checking pod events for details...")
					// Check the event log for the pods associated with this 
					//   job.
					// There may be something useful in there.
					pods, err := kubeutil.GetKubectl(kubectl, "get", "pods", "-l", kubeJobSelector, "-o", "template={{range .items}}{{.metadata.name}}{{end}}")
					if err != nil {
						log.Warn(err)
						continue
					}

					for _, podName := range strings.Split(pods, "\n") {
						involvedObject := fmt.Sprintf("involvedObject.name=%s", podName)
						kubeutil.ExecKubectl(kubectl, "get", "events", "--field-selector", involvedObject)
					}
				}

				log.Warn("Failed to attach to logs for label ", kubeJobSelector, ". Waiting ", sleepSeconds, " seconds and trying again.")

				time.Sleep(time.Second * time.Duration(sleepSeconds))
				attempts = attempts + 1

			} else {
				ok = true
			}
		}

		return nil
	},
}
