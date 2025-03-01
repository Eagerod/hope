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

		type hypervisorBuildCopy struct {
			nHyps            int
			buildHypervisors []hypervisors.Hypervisor
			copyHypervisors  []hypervisors.Hypervisor
		}
		hypEngMap := map[string]hypervisorBuildCopy{}
		for _, hypervisorName := range vm.Hypervisors {
			hypervisor, err := utils.GetHypervisor(hypervisorName)
			if err != nil {
				return err
			}

			hvNode, err := hypervisor.UnderlyingNode()
			if err != nil {
				return err
			}

			hbc := hypEngMap[hvNode.Engine]
			hbc.nHyps += 1
			switch hypervisor.CopyImageMode() {
			case hypervisors.CopyImageModeNone:
				hbc.buildHypervisors = append(hbc.buildHypervisors, hypervisor)
			case hypervisors.CopyImageModeToAll:
				if len(hbc.buildHypervisors) == 0 {
					hbc.buildHypervisors = append(hbc.buildHypervisors, hypervisor)
				}
				hbc.copyHypervisors = append(hbc.copyHypervisors, hypervisor)
			case hypervisors.CopyImageModeFromFirst:
				if len(hbc.buildHypervisors) == 0 {
					hbc.buildHypervisors = append(hbc.buildHypervisors, hypervisor)
				} else {
					hbc.copyHypervisors = append(hbc.copyHypervisors, hypervisor)
				}
			}
		}

		for engine, hbc := range hypEngMap {
			log.Debugf("Creating VM %s using %d %s hypervisors", vm.Name, hbc.nHyps, engine)
			for _, hv := range hbc.buildHypervisors {
				if err := hv.CreateImage(vms, *vm, *imageCmdParameterSlice, imageCmdForceFlag); err != nil {
					return err
				}
			}

			firstHV := hbc.buildHypervisors[0]
			for _, hv := range hbc.copyHypervisors {
				if err := hv.CopyImage(vms, *vm, firstHV); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
