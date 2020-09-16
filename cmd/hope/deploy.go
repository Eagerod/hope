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

		resourcesToDeploy := []Resource{}

		if len(args) == 0 {
			log.Debug("Received no arguments for deployment. Deploying all resources.")
			resourcesToDeploy = *resources
		} else {
			log.Debug("Deploying these resources: \n\t", strings.Join(args, "\n\t"), "\nIn the order given.")

			// Map the slice by name so they can be fetched in order.
			resourcesMap := map[string]Resource{}
			for _, resource := range *resources {
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


		for _, resource := range resourcesToDeploy {
			if err := hope.KubectlApplyF(kubectl, resource.File); err != nil {
				return err
			}
		}

		return nil
	},
}
