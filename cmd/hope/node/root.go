package node

import (
	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use:   "node",
	Short: "command line tool for managing nodes on the network",
	Long: "Manage ESXi vms",
}

func InitNodeCommand() {
	RootCommand.AddCommand(hostnameCmd)
	RootCommand.AddCommand(initCmd)
	RootCommand.AddCommand(resetCmd)
	RootCommand.AddCommand(sshCmd)

	initHostnameCmdFlags()
	initResetCmd()
	initSshCmd()
}
