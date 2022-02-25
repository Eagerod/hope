package cmd

import (
	"errors"
	"fmt"
	"os"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
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
		// Why not?
		if len(args) == 0 && len(*removeCmdTagSlice) == 0 {
			return errors.New("Cannot remove all resources at once")
		}

		resources, err := utils.GetIdentifiableResources(&args, removeCmdTagSlice)
		if err != nil {
			return err
		}

		if len(*resources) == 0 {
			log.Warn("No resources matched the provided definitions.")
			return nil
		}

		kubectl, err := utils.KubectlFromAnyMaster()
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

			parameters, err := utils.FlattenParameters(resource.Parameters, resource.FileParameters)
			if err != nil {
				return err
			}

			// It is possible that names of resources are created using
			//   templated values, so still do the environment substitution
			//   process.
			switch resourceType {
			case hope.ResourceTypeFile:
				if len(parameters) != 0 {
					if info, err := os.Stat(resource.File); err != nil {
						return err
					} else if info.IsDir() {
						log.Trace("Deleting directory with parameters; creating copy for parameter substitution.")
						tempDir, err := hope.ReplaceParametersInDirectoryCopy(resource.File, parameters)
						if err != nil {
							return err
						}
						defer os.RemoveAll(tempDir)

						if err := hope.KubectlDeleteF(kubectl, tempDir); err != nil {
							return err
						}
					} else {
						content, err := hope.ReplaceParametersInFile(resource.File, parameters)
						if err != nil {
							return err
						}

						if err := hope.KubectlDeleteStdIn(kubectl, content); err != nil {
							return err
						}
					}
				} else {
					log.Trace(resource.Name, " does not have any parameters. Skipping population and deleting file directly")
					if err := hope.KubectlDeleteF(kubectl, resource.File); err != nil {
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

				if err := hope.KubectlDeleteStdIn(kubectl, inline); err != nil {
					return err
				}
			case hope.ResourceTypeDockerBuild:
				log.Debug("Skipping removal of docker image.")
			case hope.ResourceTypeJob:
				log.Debug("Skipping removal of job resource type.")
			case hope.ResourceTypeExec:
				log.Debug("Skipping removal of exec resource type.")
			default:
				return errors.New(fmt.Sprintf("Resource type (%s) not implemented.", resourceType))
			}
		}

		return nil
	},
}
