package cmd

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

var kubectlCmd = &cobra.Command{
	Use:   "kubectl",
	Short: "Pull a temporary kubeconfig from the cluster, and run a kubectl command against it",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubectl, err := utils.KubectlFromAnyMaster()
		if err != nil {
			return err
		}

		defer kubectl.Destroy()

		return kubeutil.ExecKubectl(kubectl, args...)
	},
}
