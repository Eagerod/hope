package cmd

import (
	"errors"
	"fmt"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
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
		resources, err := getResources()
		if err != nil {
			return err
		}

		// TODO: Re-evaluate; maybe just deploy everything in order defined.
		if len(args) != 0 && len(*deployCmdTagSlice) != 0 {
			return errors.New("Cannot deploy tags and named resources together.")
		}

		resourcesToDeploy := []Resource{}

		if len(args) == 0 {
			if len(*deployCmdTagSlice) != 0 {
				tagMap := map[string]bool{}
				for _, tag := range *deployCmdTagSlice {
					tagMap[tag] = true
				}

				for _, resource := range *resources {
					for _, tag := range resource.Tags {
						if _, ok := tagMap[tag]; ok {
							resourcesToDeploy = append(resourcesToDeploy, resource)
							continue
						}
					}
				}

				log.Debug("Deploying these resources: \n\t", strings.Join(args, "\n\t"), "\nFrom provided tags.")
			} else {
				log.Debug("Received no arguments for deployment. Deploying all resources.")
				resourcesToDeploy = *resources
			}
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

		// Do a pass over the resources, and make sure that there's a docker
		//   build step before potentially asking the user to type in their
		//   password to elevate
		hasDockerResource := false
		for _, resource := range resourcesToDeploy {
			resourceType, _ := resource.GetType()
			if resourceType == ResourceTypeDockerBuild {
				hasDockerResource = true
				break
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

		// Wait as long as possible before pulling the temporary kubectl from
		//   a master node.
		// TODO: Implement something similar to the hasDockerResource process
		//   above; if there isn't anything that needs to talk to kubernetes,
		//   don't even bother pulling the kubeconfig.
		masters := viper.GetStringSlice("masters")
		kubectl, err := getKubectlFromAnyMaster(log.WithFields(log.Fields{}), masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		// TODO: Should be done in hope pkg
		// TODO: Add validation to ensure each type of deployment can run given
		//   the current dev environment -- ensure docker is can connect, etc.
		for _, resource := range resourcesToDeploy {
			log.Debug("Starting deployment of ", resource.Name)
			resourceType, err := resource.GetType()
			if err != nil {
				return err
			}

			switch resourceType {
			case ResourceTypeFile:
				if len(resource.Parameters) != 0 {
					content, err := replaceParametersInFile(resource.File, resource.Parameters)
					if err != nil {
						return err
					}

					if err := hope.KubectlApplyStdIn(kubectl, content); err != nil {
						return err
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population and applying file directly")
					if err := hope.KubectlApplyF(kubectl, resource.File); err != nil {
						return err
					}
				}
			case ResourceTypeInline:
				inline := resource.Inline

				// Log out the inline resource before substituting it; secrets
				//   are likely being populated.
				log.Trace(inline)

				if len(resource.Parameters) != 0 {
					inline, err = replaceParametersInString(inline, resource.Parameters)
					if err != nil {
						return err
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population.")
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
				if err := hope.FollowLogsAndPollUntilJobComplete(log.WithFields(log.Fields{}), kubectl, resource.Job, 10, 60); err != nil {
					return err
				}
			case ResourceTypeExec:
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
