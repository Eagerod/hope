package vm

import (
	"errors"
	"fmt"
	"os"
	"path"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Creates a VM image from the defined packer spec.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		
		vms, err := getVMs()
		if err != nil {
			return err
		}

		// Get the VM Spec
		var vmSpec *VMImageSpec
		for _, vm := range vms.Images {
			if vm.Name == vmName {
				vmSpec = &vm
				break
			}
		}

		if vmSpec == nil {
			return errors.New(fmt.Sprintf("No VM named %s found in images definitions.", vmName))
		}

		vmDir := path.Join(vms.RootDir, vmName)

		stat, err := os.Stat(vmDir)
		if err != nil && os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("VM spec directory (%s) not found", vmDir))
		} else if err != nil {
			return err
		}

		if !stat.IsDir() {
			return errors.New(fmt.Sprintf("VM spec directory (%s) is just a file", vmDir))
		}
		

		// Copy the directory out to a temporary director, and iterate through
		//   all the files, running text substitution against them with the
		//   list of given parameters.
		
		// Check caches to see if I event want to build this again


		log.Info("I'm going to create a VM image.")
		return nil
	},
}
