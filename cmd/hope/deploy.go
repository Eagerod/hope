package cmd

import (
	"fmt"
	"os"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/docker"
	"github.com/Eagerod/hope/pkg/helm"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var deployCmdTagSlice *[]string

func initDeployCmdFlags() {
	deployCmdTagSlice = deployCmd.Flags().StringArrayP("tag", "t", []string{}, "deploy resources with this tag")
}

var deployCmd = &cobra.Command{
	Use:   "deploy [resource-name]...",
	Short: "Deploy Kubernetes resources defined in the hope file",
	Args:  cobra.ArbitraryArgs,
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
			kubectl, err = utils.KubectlFromAnyMaster(log.WithFields(log.Fields{}))
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

			parameters, err := utils.FlattenParameters(resource.Parameters, resource.FileParameters)
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
				if err := hope.DoDockerBuild(log.WithFields(log.Fields{}), resource); err != nil {
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
			case hope.ResourceTypeHelm:
				if hasRepo, err := helm.HasRepo(resource.Helm.Repo, resource.Helm.Path); err != nil {
					return err
				} else if !hasRepo {
					if err := helm.ExecHelm("repo", "add", resource.Helm.Repo, resource.Helm.Path); err != nil {
						return err
					}
				}

				if err := helm.ExecHelm("repo", "update", resource.Helm.Repo); err != nil {
					return err
				}

				allArgs := []string{"upgrade", "--install"}
				if len(resource.Helm.Namespace) != 0 {
					allArgs = append(allArgs, "--namespace", resource.Helm.Namespace, "--create-namespace")
				}

				if len(resource.Helm.ValuesFile) != 0 {
					log.Trace("Copying values file for parameter replacement")
					tempFile, err := hope.ReplaceParametersInFileCopy(resource.Helm.ValuesFile, parameters)
					if err != nil {
						return err
					}
					defer os.Remove(tempFile)

					allArgs = append(allArgs, "--values", tempFile)
				}

				if len(resource.Helm.Version) != 0 {
					allArgs = append(allArgs, "--version", resource.Helm.Version)
				}

				allArgs = append(allArgs, resource.Helm.Release, resource.Helm.Chart)
				if err := helm.ExecHelm(allArgs...); err != nil {
					return err
				}
			default:
				return fmt.Errorf("resource type (%s) not implemented", resourceType)
			}
		}

		return nil
	},
}
