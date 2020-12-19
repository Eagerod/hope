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
	ipCmd.Flags().IntVarP(&ipCmdNumRetries, "retries", "r", 10, "how many reties before failing the IP command (default 10).")
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
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		if ipCmdNumRetries == 0 {
			return errors.New("Cannot make 0 attempts to fetch IP address")
		}

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("Node %s is not a hypervisor; cannot find a node's IP from it", hypervisor.Name)
		}

		log.Tracef("Will attempt to fetch IP address %d times", ipCmdNumRetries)

		// This currently makes two calls over SSH per loop, rather than a
		//   single request to grab the VM's world id, and then only one call
		//   per loop to look up the IP address.
		sleepSeconds := time.Duration(1)
		for ; ipCmdNumRetries > 0; ipCmdNumRetries-- {
			ip, err := esxi.GetIpAddressOfVmNamed(hypervisor.ConnectionString(), vmName)
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
