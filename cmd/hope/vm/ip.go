package vm

import (
	"errors"
	"fmt"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
)

var ipCmdNumRetries int

func initIpCmdFlags() {
	ipCmd.Flags().IntVarP(&ipCmdNumRetries, "retries", "r", 10, "how many reties before failing the IP command.")
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

var ipCmd = &cobra.Command{
	Use:   "ip <node-name>",
	Short: "Get the IP address of a VM on the specified hypervisor.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		if ipCmdNumRetries <= 0 {
			return errors.New("cannot make 0 attempts to fetch IP address")
		}

		hypervisor, err := utils.HypervisorForNodeNamed(vmName)
		if err != nil {
			return err
		}

		log.Tracef("Will attempt to fetch IP address %d times", ipCmdNumRetries)

		// This currently makes two calls over SSH per loop, rather than a
		//   single request to grab the VM's world id, and then only one call
		//   per loop to look up the IP address.
		sleepDuration := time.Duration(1)
		for ; ipCmdNumRetries > 0; ipCmdNumRetries-- {
			ip, err := (*hypervisor).VMIPAddress(vmName)
			if err == nil {
				fmt.Println(ip)
				return nil
			}

			log.Debugf("VM hasn't bound an IP address yet. Waiting %d seconds before checking again...", sleepDuration)
			time.Sleep(sleepDuration * time.Second)
			sleepDuration = minDuration(sleepDuration*2, 10)
		}

		return nil
	},
}
