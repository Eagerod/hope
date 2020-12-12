package vm

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/pkg/esxi"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		vmName := args[1]

		return esxi.PowerOffVmNamed(host, vmName)
	},
}
