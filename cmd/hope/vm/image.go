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

		// Group hypervisors by engine
		hypEngMap := map[string][]hypervisors.Hypervisor{}
		for _, hypervisorName := range vm.Hypervisors {
			hypervisor, err := utils.GetHypervisor(hypervisorName)
			if err != nil {
				return err
			}

			hvNode, err := hypervisor.UnderlyingNode()
			if err != nil {
				return err
			}

			engHVs := hypEngMap[hvNode.Engine]
			engHVs = append(engHVs, hypervisor)
			hypEngMap[hvNode.Engine] = engHVs
		}

		for engine, engHypervisors := range hypEngMap {
			log.Debugf("Creating VM %s using %d %s hypervisors", vm.Name, len(vm.Hypervisors), engine)
			firstHV := engHypervisors[0]

			packerSpec, err := firstHV.CreateImage(vms, *vm, *imageCmdParameterSlice, imageCmdForceFlag)
			if err != nil {
				return err
			}

			for _, hv := range engHypervisors {
				if err := hv.CopyImage(*packerSpec, vms, *vm); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
