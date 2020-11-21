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
	"github.com/Eagerod/hope/pkg/hope"
)

var removeCmdTagSlice *[]string

func initRemoveCmdFlags() {
	removeCmdTagSlice = removeCmd.Flags().StringArrayP("tag", "t", []string{}, "remove resources with this tag")
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Kubernetes resources defined in the hope file",
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := getResources()
		if err != nil {
			return err
		}

		// TODO: Re-evaluate; maybe just remove everything in order defined.
		if len(args) != 0 && len(*removeCmdTagSlice) != 0 {
			return errors.New("Cannot removed tags and named resources together.")
		}

		// Why not?
		if len(args) == 0 && len(*removeCmdTagSlice) == 0 {
			return errors.New("Cannot remove all resources at once.")
		}

		resourcesToRemove := []Resource{}

		if len(args) == 0 {
			if len(*removeCmdTagSlice) != 0 {
				tagMap := map[string]bool{}
				for _, tag := range *removeCmdTagSlice {
					tagMap[tag] = true
				}

				resourceNames := []string{}
				for _, resource := range *resources {
					for _, tag := range resource.Tags {
						if _, ok := tagMap[tag]; ok {
							resourcesToRemove = append(resourcesToRemove, resource)
							resourceNames = append(resourceNames, resource.Name)
							continue
						}
					}
				}

				log.Debug("Removing these resources: \n\t", strings.Join(resourceNames, "\n\t"), "\nFrom provided tags.")
			}
		} else {
			log.Debug("Removing these resources: \n\t", strings.Join(args, "\n\t"), "\nIn reverse order given.")

			// Map the slice by name so they can be fetched in order.
			resourcesMap := map[string]Resource{}
			for _, resource := range *resources {
				_, ok := resourcesMap[resource.Name]
				if ok {
					return errors.New(fmt.Sprintf("Multiple resources found with name %s. Aborting removal", resource.Name))
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

				resourcesToRemove = append(resourcesToRemove, resource)
			}
		}

		if len(resourcesToRemove) == 0 {
			log.Warn("No resources matched the provided definitions.")
			return nil
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
		for i := len(resourcesToRemove)-1; i >= 0; i-- {
			resource := resourcesToRemove[i]
			log.Debug("Starting removal of ", resource.Name)
			resourceType, err := resource.GetType()
			if err != nil {
				return err
			}

			// It is possible that names of resources are created using
			//   templated values, so still do the environment substitution
			//   process.
			switch resourceType {
			case ResourceTypeFile:
				if len(resource.Parameters) != 0 {
					content, err := replaceParametersInFile(resource.File, resource.Parameters)
					if err != nil {
						return err
					}

					if err := hope.KubectlDeleteStdIn(kubectl, content); err != nil {
						return err
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population and applying file directly")
					if err := hope.KubectlDeleteF(kubectl, resource.File); err != nil {
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

				if err := hope.KubectlDeleteStdIn(kubectl, inline); err != nil {
					return err
				}
			case ResourceTypeDockerBuild:
				log.Debug("Skipping removal of docker image.")
			case ResourceTypeJob:
				log.Debug("Skipping removal of job resource type.")
			case ResourceTypeExec:
				log.Debug("Skipping removal of exec resource type.")
			default:
				return errors.New(fmt.Sprintf("Resource type (%s) not implemented.", resourceType))
			}
		}

		return nil
	},
}
