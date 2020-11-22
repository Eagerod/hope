package cmd

import (
	"errors"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var removeCmdTagSlice *[]string

func initRemoveCmdFlags() {
	removeCmdTagSlice = removeCmd.Flags().StringArrayP("tag", "t", []string{}, "remove resources with this tag")
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Kubernetes resources defined in the hope file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Why not?
		if len(args) == 0 && len(*removeCmdTagSlice) == 0 {
			return errors.New("Cannot remove all resources at once.")
		}

		resources, err := getIdentifiableResources(&args, removeCmdTagSlice)
		if err != nil {
			return err
		}

		if len(*resources) == 0 {
			log.Warn("No resources matched the provided definitions.")
			return nil
		}

		// Wait as long as possible before pulling the temporary kubectl from
		//   a master node.
		masters := viper.GetStringSlice("masters")
		kubectl, err := kubeutil.NewKubectlFromAnyNode(masters)
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		// TODO: Should be done in hope pkg
		for i := len(*resources) - 1; i >= 0; i-- {
			resource := (*resources)[i]
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
