package node

import (
	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use:   "node",
	Short: "manage nodes on the network",
	Long:  "Manage adding, removing, modifying nodes in the cluster",
}

func InitNodeCommand() {
	RootCommand.AddCommand(hostnameCmd)
	RootCommand.AddCommand(initCmd)
	RootCommand.AddCommand(listCmd)
	RootCommand.AddCommand(resetCmd)
	RootCommand.AddCommand(sshCmd)
	RootCommand.AddCommand(validateCmd)

	initHostnameCmdFlags()
	initInitCmd()
	initListCmd()
	initResetCmd()
	initValidateCmd()
}
