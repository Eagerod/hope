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

		var hypervisorBuildCopy = struct {
			engine           string
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

			hbp := hypEngMap[hvNode.Engine]
			hbp.nHyps += 1
			switch hypervisor.CopyImageMode() {
			case hypervisors.CopyImageModeNone:
				hbp.buildHypervisors = append(hbp.buildHypervisors, hypervisor)
			case hypervisors.CopyImageModeToAll:
				if len(hbp.buildHypervisors) == 0 {
					hbp.buildHypervisors = append(hbp.buildHypervisors, hypervisor)
				}
				hbp.copyHypervisors = append(hbp.copyHypervisors, hypervisor)
			case hypervisors.CopyImageModeFromFirst:
				if len(hbp.buildHypervisors) == 0 {
					hbp.buildHypervisors = append(hbp.buildHypervisors, hypervisor)
				} else {
					hbp.copyHypervisors = append(hbp.copyHypervisors, hypervisor)
				}
			}
		}

		for engine, hyps := range hypEngMap {
			log.Debugf("Creating VM %s using %d %s hypervisors", vm.Name, hbp.nHyps, engine)
			for _, hv := range hyps.buildHypervisors {
				if err := hv.CreateImage(vms, *vm, *imageCmdParameterSlice, imageCmdForceFlag); err != nil {
					return err
				}
			}

			firstHV := hyps.buildHypervisors[0]
			for _, hv := range hyps.copyHypervisors {
				if err := hv.CopyImage(vms, *vm, firstHV); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
