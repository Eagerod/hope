package vm

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/pkg/esxi"
)

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Get the IP address of a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		vmName := args[1]

		ip, err := esxi.GetIpAddressOfVmNamed(host, vmName)
		if err != nil {
			return err
		}

		fmt.Println(ip)

		return nil
	},
}
