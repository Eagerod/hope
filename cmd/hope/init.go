package cmd

import (
	"errors"
	"fmt"
)

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap the master node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("This command will attempt to bootstrap a master node.")

		masterIp := args[0]
		masterExpected := stringInSlice(masterIp, viper.GetStringSlice("masters"))

		if !masterExpected {
			return errors.New(fmt.Sprintf("Failed to find master %s in config", masterIp))
		}

		fmt.Println("Running some tests to ensure this process can be run properly...")

		if err := ssh.TestCanSSH(masterIp); err != nil {
			// Try to recover this.
			if err = ssh.TryConfigureSSH(masterIp); err != nil {
				return err
			}

			fmt.Println("Configured passwordless SSH using the identity file that SSH uses for this connection by default")
		}

		fmt.Println("Creating cluster at", masterIp)

		return nil
	},
}

func stringInSlice(value string, slc []string) bool {
	for _, s := range slc {
		if s == value {
			return true
		}
	}

	return false
}
