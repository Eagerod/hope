package unifi

import (
	"fmt"
	"regexp"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/sliceutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

var apsCmd = &cobra.Command{
	Use:   "ap <access-point-ip>",
	Short: "Initialize an AP by setting its inform URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apIp := args[0]
		aps := viper.GetStringSlice("access_points")

		if !sliceutil.StringInSlice(apIp, aps) {
			return fmt.Errorf("failed to find %s in access points list", apIp)
		}

		apController := viper.GetString("access_point_controller")
		informAddress := fmt.Sprintf("%s/inform", apController)

		// Should move this to a pkg
		infoArgs := []string{apIp, "mca-cli-op", "info"}
		output, err := ssh.GetSSH(infoArgs...)
		if err != nil {
			return err
		}

		informAddressStr := "Status:\\s*Connected \\(([^\\)]+)\\)"
		informAddressRegexp, err := regexp.Compile(informAddressStr)
		if err != nil {
			return err
		}

		currentInformAddress := informAddressRegexp.FindStringSubmatch(output)
		if len(currentInformAddress) < 2 {
			log.Infof("Access point %s is current not connected to a controller.", apIp)
		} else if currentInformAddress[1] == informAddress {
			log.Infof("Access point %s already configured correctly to inform at %s", apIp, informAddress)
			return nil
		} else {
			log.Infof("Access point %s is currently connected to %s", apIp, currentInformAddress[1])
		}

		allArgs := []string{apIp, "mca-cli-op", "set-inform", informAddress}
		return ssh.ExecSSH(allArgs...)
	},
}
