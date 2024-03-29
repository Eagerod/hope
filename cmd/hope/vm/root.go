package vm

import (
	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use:   "vm",
	Short: "manages vm resources on the network",
	Long:  "Manage creating templates, starting nodes, and relocating nodes.",
}

func InitVMCommand() {
	RootCommand.AddCommand(createCmd)
	RootCommand.AddCommand(deleteCmd)
	RootCommand.AddCommand(imageCmd)
	RootCommand.AddCommand(ipCmd)
	RootCommand.AddCommand(listCmd)
	RootCommand.AddCommand(startCmd)
	RootCommand.AddCommand(stopCmd)

	initImageCmdFlags()
	initIpCmdFlags()
}
