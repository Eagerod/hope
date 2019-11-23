package cmd

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Bootstrap a worker node",
	RunE: func(cmd *cobra.Command, args []string) error {
    	fmt.Println("I'm going to install a bunch of dependencies, and turn this into a worker node...")
		return nil
	},
}
