package cmd

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/docker"
	"github.com/Eagerod/hope/pkg/envsubst"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

const MaximumJobDeploymentPollSeconds int = 60

// rootCmd represents the base command when called without any subcommands
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a Kubernetes yaml file",
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := getResources()
		if err != nil {
			return err
		}

		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		resourcesToDeploy := []Resource{}

		if len(args) == 0 {
			log.Debug("Received no arguments for deployment. Deploying all resources.")
			resourcesToDeploy = *resources
		} else {
			log.Debug("Deploying these resources: \n\t", strings.Join(args, "\n\t"), "\nIn the order given.")

			// Map the slice by name so they can be fetched in order.
			resourcesMap := map[string]Resource{}
			for _, resource := range *resources {
				_, ok := resourcesMap[resource.Name]
				if ok {
					return errors.New(fmt.Sprintf("Multiple resources found with name %s. Aborting deploy", resource.Name))
				}

				resourcesMap[resource.Name] = resource
			}

			// Do an initial pass to ensure that no invalid resources were
			//   provided
			for _, expectedResource := range args {
				resource, ok := resourcesMap[expectedResource]
				if !ok {
					return errors.New(fmt.Sprintf("Cannot find resource '%s' in configuration file.", expectedResource))
				}

				resourcesToDeploy = append(resourcesToDeploy, resource)
			}
		}

		// TODO: Should be done in hope pkg
		// TODO: Add validation to ensure each type of deployment can run given
		//   the current dev environment -- ensure docker is can connect, etc.
		for _, resource := range resourcesToDeploy {
			resourceType, err := resource.GetType()
			if err != nil {
				return err
			}

			switch resourceType {
			case ResourceTypeFile:
				if err := hope.KubectlApplyF(kubectl, resource.File); err != nil {
					return err
				}
			case ResourceTypeInline:
				inline := resource.Inline
				inline, err := envsubst.GetEnvsubst(inline)
				if err != nil {
					return err
				}

				if err := hope.KubectlApplyStdIn(kubectl, inline); err != nil {
					return err
				}
			case ResourceTypeDockerBuild:
				// Strip the actual tag off the repo so that it defaults to the
				//   latest.
				tagSeparator := strings.LastIndex(resource.Build.Tag, ":")
				pullImage := resource.Build.Tag
				if tagSeparator != -1 {
					pullImage = pullImage[:tagSeparator]
				}

				if err := docker.ExecDocker("pull", pullImage); err != nil {
					// Maybe the image was pushed with the given tag.
					// Maybe the tag is something like :stable.
					// Hopefully we can grab a few layers at least.
					if err := docker.ExecDocker("pull", resource.Build.Tag); err != nil {
						log.Warn("Failed to pull existing images for ", pullImage, ". Maybe this image doesn't exist?")

						// Don't return any errors here.
						// If this is the first time this image is being
						//   pushed, there will be nothing to pull, and
						//   this will never succeed.
					}
				}
				if err := docker.ExecDocker("build", resource.Build.Path, "-t", resource.Build.Tag); err != nil {
					return err
				}
				if err := docker.ExecDocker("push", resource.Build.Tag); err != nil {
					return err
				}
			case ResourceTypeJob:
				// Exponential backoff maxing out at 60 seconds.
				// TODO: Implement maximum retries, or other throughput-related
				//   controls
				// TODO: Fetch more detailed job status information to show on
				//   the console.
				attempts := 1
				jobLogger := log.WithFields(log.Fields{})
				for ok := false; !ok; {
					status, err := hope.GetJobStatus(jobLogger, kubectl, resource.Job)
					if err != nil {
						return err
					}

					switch status {
					case hope.JobStatusFailed:
						return errors.New(fmt.Sprintf("Job %s failed.", resource.Job))
					case hope.JobStatusComplete:
						log.Debug("Job ", resource.Job, " successful.")
						ok = true
						break
					default:
						// If the job is running, start polling for logs.
						// Jobs that failed or completed long in the past may
						//   have had their pods wiped since they ran.
						if err := hope.FollowLogsIfContainersRunning(kubectl, resource.Job); err != nil {
							log.Warn(err)
							attemptsDuration := math.Pow(2, float64(attempts-1))
							sleepSeconds := int(math.Min(attemptsDuration, float64(MaximumJobDeploymentPollSeconds)))
					
							if sleepSeconds == MaximumJobDeploymentPollSeconds {
								log.Debug("Checking pod events for details...")
								// Check the event log for the pods associated
								//   with this job.
								// There may be something useful in there.
								pods, err := hope.GetPodsForJob(kubectl, resource.Job)
								if err != nil {
									log.Warn(err)
									continue
								}
					
								for _, pods := range *pods {
									involvedObject := fmt.Sprintf("involvedObject.name=%s", pods)
									kubeutil.ExecKubectl(kubectl, "get", "events", "--field-selector", involvedObject)
								}
							}
					
							log.Warn("Failed to attach to logs for job ", resource.Job, ". Waiting ", sleepSeconds, " seconds and trying again.")
					
							time.Sleep(time.Second * time.Duration(sleepSeconds))
							attempts = attempts + 1
						}
					}
				}
			default:
				return errors.New(fmt.Sprintf("Resource type (%s) not implemented.", resourceType))
			}
		}

		return nil
	},
}
