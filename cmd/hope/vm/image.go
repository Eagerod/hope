package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
)

var imageCmdParameterSlice *[]string
var imageCmdForceFlag bool

func initImageCmdFlags() {
	imageCmd.Flags().BoolVarP(&imageCmdForceFlag, "force", "f", false, "remove existing image if one already exists")

	imageCmdParameterSlice = imageCmd.Flags().StringArrayP("param", "p", []string{}, "parameters to forward to packer's -var")
}

var imageCmd = &cobra.Command{
	Use:   "image <image-name>",
	Short: "Creates a VM image from the defined packer spec.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		vms, err := utils.GetVMs()
		if err != nil {
			return err
		}

		vm, err := utils.VMSpec(vmName)
		if err != nil {
			return err
		}

		// Each image that's made will be copied to all hypervisors that
		//   accept that image.
		hypervisors, err := utils.GetHypervisors()
		if err != nil {
			return err
		}

		log.Debugf("Creating VM %s using %d hypervisors", vm.Name, len(vm.Hypervisors))
		for _, hypervisorName := range vm.Hypervisors {
			hypervisor, err := utils.GetHypervisor(hypervisorName)
			if err != nil {
				return err
			}

			packerSpec, err := hypervisor.CreateImage(vms, *vm, *imageCmdParameterSlice, imageCmdForceFlag)
			if err != nil {
				return err
			}

			if packerSpec == nil {
				continue
			}

			for _, hv := range hypervisors {
				if err := hv.CopyImage(*packerSpec, vms, *vm); err != nil {
					return err
				}
			}

		}

		return nil
	},
}
