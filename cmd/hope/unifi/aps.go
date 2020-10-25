package unifi

import (
	"errors"
	"fmt"
)

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/sliceutil"
)

var apsCmd = &cobra.Command{
	Use:   "ap",
	Short: "Initialize an AP by setting its inform URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apIp := args[0]
		aps := viper.GetStringSlice("access_points")

		if !sliceutil.StringInSlice(apIp, aps) {
			return errors.New(fmt.Sprintf("Failed to find %s in access points list.", apIp))
		}

		fmt.Println("doing the thing")

		return nil
	},
}
