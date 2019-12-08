package cmd

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap the master node",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("I'm going to install a bunch of dependencies, and turn this into a control plane node...")
		return nil
	},
}
