package vm

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/esxi"
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
	Use:   "ip",
	Short: "Get the IP address of a VM on the specified hypervisor.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		if ipCmdNumRetries == 0 {
			return errors.New("Cannot make 0 attempts to fetch IP address")
		}

		hypervisor, err := utils.HypervisorForNodeNamed(vmName)
		if err != nil {
			return err
		}

		hypervisorNode, err := (*hypervisor).UnderlyingNode()
		if err != nil {
			return err
		}

		log.Tracef("Will attempt to fetch IP address %d times", ipCmdNumRetries)

		// This currently makes two calls over SSH per loop, rather than a
		//   single request to grab the VM's world id, and then only one call
		//   per loop to look up the IP address.
		sleepSeconds := time.Duration(1)
		for ; ipCmdNumRetries > 0; ipCmdNumRetries-- {
			ip, err := esxi.GetIpAddressOfVmNamed(hypervisorNode.ConnectionString(), vmName)
			if err != nil {
				return err
			}

			ip = strings.TrimSpace(ip)
			if ip != "0.0.0.0" {
				fmt.Println(ip)
				break
			}

			log.Debugf("VM hasn't bound an IP address yet. Waiting %d seconds before checking again...", sleepSeconds)
			time.Sleep(sleepSeconds * time.Second)
			sleepSeconds = minDuration(sleepSeconds*2, 10)
		}

		return nil
	},
}
