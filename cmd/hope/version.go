package cmd

import (
	"fmt"
	"os"
)

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version number and exits",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Print straight to console, since log level shouldn't dictate
		//   whether or not this makes it to console.
		fmt.Println(os.Args[0] + ": " + VersionBuild)

		return nil
	},
}
