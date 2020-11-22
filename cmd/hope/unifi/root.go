package unifi

import (
	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use:   "unifi",
	Short: "command line tool for managing UniFi resources",
	Long: "Manage UniFi software configurations without manually operating against the resources",
}

func InitUnifiCommand() {
	RootCommand.AddCommand(apsCmd)
}
