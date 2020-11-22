package cmd

import (
	"errors"
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

var listCmdTagSlice *[]string

func initListCmdFlags() {
	listCmdTagSlice = listCmd.Flags().StringArrayP("tag", "t", []string{}, "list resources with this tag")
}

// This whole command was pretty well ripped from the deploy command.
// Probably worth breaking it up into utils + pkg at some point.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources that belong to a particular set of tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := getResources()
		if err != nil {
			return err
		}

		resourcesToList := []Resource{}

		hasTaggedResources := false
		hasDirectResources := false
		if len(*listCmdTagSlice) != 0 {
			hasTaggedResources = true
			tagMap := map[string]bool{}
			for _, tag := range *listCmdTagSlice {
				tagMap[tag] = true
			}

			resourceNames := []string{}
			for _, resource := range *resources {
				for _, tag := range resource.Tags {
					if _, ok := tagMap[tag]; ok {
						resourcesToList = append(resourcesToList, resource)
						resourceNames = append(resourceNames, resource.Name)
						continue
					}
				}
			}
		}

		if len(args) != 0 {
			hasDirectResources = true
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

				resourcesToList = append(resourcesToList, resource)
			}
		}

		if !hasDirectResources && !hasTaggedResources {
			resourcesToList = *resources
		}

		for _, resource := range resourcesToList {
			fmt.Println(resource.Name)
		}

		return nil
	},
}
