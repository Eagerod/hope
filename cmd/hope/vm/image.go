package vm

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope/hypervisors"
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

		vmHypervisors := []hypervisors.Hypervisor{}
		for _, s := range vm.Hypervisors {
			hyp, err := utils.GetHypervisor(s)
			if err != nil {
				return err
			}

			vmHypervisors = append(vmHypervisors, hyp)
		}

		plans, err := hypervisors.GetEnginePlans(vmHypervisors)
		if err != nil {
			return err
		}

		for _, plan := range plans {
			log.Debugf("Creating VM %s using %d %s hypervisors", vm.Name, plan.NumHypervisors, plan.Engine)
			for _, hv := range plan.BuildHypervisors {
				hvNode, err := hv.UnderlyingNode()
				if err != nil {
					return err
				}

				if !imageCmdForceFlag {
					log.Tracef("Searching for %s on %s before attempting build...", vm.Name, hvNode.Name)
					hvHasBuiltImage, err := hypervisors.HasBuiltImage(hv, vms, vm.Name)
					if err != nil {
						return err
					}

					if hvHasBuiltImage {
						log.Infof("Hypervisor %s already has built image: %s, skipping image creation", hvNode.Name, vm.Name)
						continue
					}
				}

				log.Infof("Beginning build of %s on %s", vm.Name, hvNode.Name)
				if err := hv.CreateImage(vms, *vm, *imageCmdParameterSlice, imageCmdForceFlag); err != nil {
					return err
				}
			}

			firstHV := plan.BuildHypervisors[0]
			for _, hv := range plan.CopyHypervisors {
				hvNode, err := hv.UnderlyingNode()
				if err != nil {
					return err
				}

				if !imageCmdForceFlag {
					log.Tracef("Searching for %s on %s before attempting copy...", vm.Name, hvNode.Name)
					hvHasAvailableImage, err := hypervisors.HasAvailableImage(hv, vms, vm.Name)
					if err != nil {
						return err
					}

					if hvHasAvailableImage {
						log.Infof("Hypervisor %s already has available image: %s, skipping image copy", hvNode.Name, vm.Name)
						continue
					}
				}

				if err := hv.CopyImage(vms, *vm, firstHV); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
