package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/docker"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var deployCmdTagSlice *[]string

func initDeployCmdFlags() {
	deployCmdTagSlice = deployCmd.Flags().StringArrayP("tag", "t", []string{}, "deploy resources with this tag")
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Kubernetes resources defined in the hope file",
	RunE: func(cmd *cobra.Command, args []string) error {
		var resources *[]hope.Resource

		if len(args) == 0 && len(*deployCmdTagSlice) == 0 {
			r, err := utils.GetResources()
			if err != nil {
				return err
			}

			resources = r
			log.Trace("Received no arguments for deployment. Deploying all resources.")
		} else {
			r, err := utils.GetIdentifiableResources(&args, deployCmdTagSlice)
			if err != nil {
				return err
			}

			resources = r
		}

		if len(*resources) == 0 {
			log.Warn("No resources matched the provided definitions.")
			return nil
		}

		// Do a pass over the resources to be deployed, and determine what
		//   kinds of local operations need to be done before all of these
		//   things can be deployed.
		hasDockerResource := false
		hasKubernetesResource := false
		for _, resource := range *resources {
			resourceType, _ := resource.GetType()
			switch resourceType {
			case hope.ResourceTypeDockerBuild:
				hasDockerResource = true
			case hope.ResourceTypeFile, hope.ResourceTypeInline, hope.ResourceTypeJob, hope.ResourceTypeExec:
				hasKubernetesResource = true
			}
		}

		if hasDockerResource {
			docker.SetUseSudo()
			if docker.UseSudo {
				log.Info("Docker needs sudo to continue. Checking if elevated permissions are available...")
				err := docker.AskSudo()
				if err != nil {
					return err
				}
			}
		}

		var kubectl *kubeutil.Kubectl
		if hasKubernetesResource {
			var err error
			kubectl, err = utils.KubectlFromAnyMaster()
			if err != nil {
				return err
			}

			defer kubectl.Destroy()
		}

		// TODO: Should be done in hope pkg
		// TODO: Add validation to ensure each type of deployment can run given
		//   the current dev environment -- ensure docker can connect, etc.
		for _, resource := range *resources {
			log.Debug("Starting deployment of ", resource.Name)
			resourceType, err := resource.GetType()
			if err != nil {
				return err
			}

			parameters, err := utils.RenderParameters(resource.Parameters, resource.FileParameters)
			if err != nil {
				return err
			}

			switch resourceType {
			case hope.ResourceTypeFile:
				if len(parameters) != 0 {
					if info, err := os.Stat(resource.File); err != nil {
						return err
					} else if info.IsDir() {
						log.Trace("Deploying directory with parameters; creating copy for parameter substitution.")
						tempDir, err := hope.ReplaceParametersInDirectoryCopy(resource.File, parameters)
						if err != nil {
							return err
						}
						defer os.RemoveAll(tempDir)

						if err := hope.KubectlApplyF(kubectl, tempDir); err != nil {
							return err
						}
					} else {
						content, err := hope.ReplaceParametersInFile(resource.File, parameters)
						if err != nil {
							return err
						}

						if err := hope.KubectlApplyStdIn(kubectl, content); err != nil {
							return err
						}
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population and applying file directly")
					if err := hope.KubectlApplyF(kubectl, resource.File); err != nil {
						return err
					}
				}
			case hope.ResourceTypeInline:
				inline := resource.Inline

				// Log out the inline resource before substituting it; secrets
				//   are likely being populated.
				log.Trace(inline)

				if len(parameters) != 0 {
					inline, err = hope.ReplaceParametersInString(inline, parameters)
					if err != nil {
						return err
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population.")
				}

				if err := hope.KubectlApplyStdIn(kubectl, inline); err != nil {
					return err
				}
			case hope.ResourceTypeDockerBuild:
				isCacheCommand := len(resource.Build.Source) != 0
				isBuildCommand := len(resource.Build.Path) != 0

				if isCacheCommand && isBuildCommand {
					return errors.New(fmt.Sprintf("Docker build step %s cannot have a path and a source", resource.Name))
				}

				// TODO: Move these to constants somewhere
				pullConstraintAlways := resource.Build.Pull == "always"
				pullConstraintIfNotPresent := resource.Build.Pull == "if-not-present" || resource.Build.Pull == ""

				if !pullConstraintAlways && !pullConstraintIfNotPresent {
					return errors.New(fmt.Sprintf("Unknown Docker image pull constraint: %s", resource.Build.Pull))
				}

				pullImage := ""
				if isCacheCommand {
					pullImage = resource.Build.Source
				} else {
					pullImage = resource.Build.Tag
				}

				ifNotPresentShouldPull := false
				if pullConstraintIfNotPresent {
					output, err := docker.GetDocker("images", pullImage, "--format={{.Repository}}:{{.Tag}}")
					if err != nil {
						return err
					}

					outputLines := strings.Split(output, "\n")
					if len(outputLines) == 0 {
						log.Info(fmt.Sprintf("No Docker images like %s not found locally, must pull from upstream.", pullImage))
						ifNotPresentShouldPull = true
					} else {
						// Figure out if the latest tag needs to be defaulted
						//   to, or if a specific one was requested.
						searchTag := pullImage
						tagIndex := strings.LastIndex(searchTag, ":")
						if tagIndex == -1 {
							log.Debug("Provided image isn't tagged; assuming latest")
							searchTag = fmt.Sprintf("%s:latest", searchTag)
						}

						log.Trace(fmt.Sprintf("Searching for local copy of tag: %s", searchTag))

						imageFound := false
						for _, imageTag := range outputLines {
							if imageTag == searchTag {
								log.Debug(fmt.Sprintf("Docker image matching %s found, skipping upstream pull", searchTag))
								imageFound = true
								break
							}
						}

						if !imageFound {
							log.Info(fmt.Sprintf("Docker image %s not found among candidates, must pull from upstream", searchTag))
							ifNotPresentShouldPull = true
						}
					}
				}

				if ifNotPresentShouldPull || pullConstraintAlways {
					if err := docker.ExecDocker("pull", pullImage); err != nil {
						return errors.New(fmt.Sprintf("Failed to find image named %s", pullImage))
					}
				}

				if isBuildCommand {
					if err := docker.ExecDocker("build", resource.Build.Path, "-t", resource.Build.Tag); err != nil {
						return err
					}
				} else {
					if err := docker.ExecDocker("tag", resource.Build.Source, resource.Build.Tag); err != nil {
						return err
					}
				}

				if err := docker.ExecDocker("push", resource.Build.Tag); err != nil {
					return err
				}
			case hope.ResourceTypeJob:
				if err := hope.FollowLogsAndPollUntilJobComplete(log.WithFields(log.Fields{}), kubectl, resource.Job, 10, 60); err != nil {
					return err
				}
			case hope.ResourceTypeExec:
				allArgs := []string{"exec", "-it", resource.Exec.Selector}
				if len(resource.Exec.Timeout) != 0 {
					allArgs = append(allArgs, "--pod-running-timeout", resource.Exec.Timeout)
				}

				allArgs = append(allArgs, "--")
				allArgs = append(allArgs, resource.Exec.Command...)

				if err := kubeutil.ExecKubectl(kubectl, allArgs...); err != nil {
					return err
				}
			default:
				return errors.New(fmt.Sprintf("Resource type (%s) not implemented.", resourceType))
			}
		}

		return nil
	},
}
