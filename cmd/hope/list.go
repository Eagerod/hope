package cmd

import (
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
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
		var resources *[]hope.Resource

		if len(args) == 0 && len(*listCmdTagSlice) == 0 {
			r, err := utils.GetResources()
			if err != nil {
				return err
			}

			resources = r
			log.Trace("Received no arguments for list. Listing all resources.")
		} else {
			r, err := utils.GetIdentifiableResources(&args, listCmdTagSlice)
			if err != nil {
				return err
			}

			resources = r
		}

		for _, resource := range *resources {
			fmt.Println(resource.Name)
		}

		return nil
	},
}
