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
	"github.com/Eagerod/hope/pkg/envsubst"
	"github.com/Eagerod/hope/pkg/hope"
)

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
			default:
				return errors.New(fmt.Sprintf("Resource type unknown. Check %s for issues", resource.Name))
			}
		}

		return nil
	},
}
