package vm

import (
	"fmt"
	"os"
	"path"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/ssh"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates the named node as a VM using its defined hypervisor.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		nodeName := args[1]

		node, err := utils.GetBareNode(nodeName)
		if err != nil {
			return err
		}

		hypervisor, err := utils.GetHypervisor(node.Hypervisor)
		if err != nil {
			return err
		}

		vms, err := utils.GetVMs()
		if err != nil {
			return err
		}

		var vm hope.VMImageSpec
		for _, aVm := range vms.Images {
			if aVm.Name == vmName {
				vm = aVm
				break
			}
		}

		vmDir := path.Join(vms.Root, vm.Name)

		log.Debug(fmt.Sprintf("Copying contents of %s for parameter replacement.", vmDir))
		tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, vm.Parameters)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		tempPackerJsonPath := path.Join(tempDir, "packer.json")
		packerSpec, err := packer.SpecFromPath(tempPackerJsonPath)
		if err != nil {
			return err
		}

		// Exec OVF tool to start VM.
		// https://www.virtuallyghetto.com/2012/05/how-to-deploy-ovfova-in-esxi-shell.html
		sourceNetworkName, ok := packerSpec.Builders[0].VMXData["ethernet0.networkName"]
		if !ok {
			return fmt.Errorf("Failed to find network definition in VM spec: %s", vmName)
		}

		hypervisorNode, err := hypervisor.UnderlyingNode()
		if err != nil {
			return err
		}

		datastoreRoot := path.Join("/", "vmfs", "volumes", hypervisorNode.Datastore)
		vmOvfName := fmt.Sprintf("%s.ovf", packerSpec.Builders[0].VMName)
		remoteOvfPath := path.Join(datastoreRoot, "ovfs", packerSpec.Builders[0].VMName, vmOvfName)
		allArgs := []string{
			hypervisorNode.ConnectionString(),
			path.Join(datastoreRoot, "bin", "ovftool", "ovftool"),
			"--diskMode=thin",
			fmt.Sprintf("--datastore=%s", hypervisorNode.Datastore),
			fmt.Sprintf("--name=%s", node.Name),
			fmt.Sprintf("--net:'%s=%s'", sourceNetworkName, hypervisorNode.Network),
			fmt.Sprintf("--numberOfCpus:'*'=%d", node.Cpu),
			fmt.Sprintf("--memorySize:'*'=%d", node.Memory),
		}

		allArgs = append(allArgs, remoteOvfPath, "vi://root@localhost")

		// Check to see if the ESXI_ROOT_PASSWORD environment if set.
		// If so, pass it on to the ssh invocation to help limit user
		//   interaction.
		esxiRootPassword := os.Getenv("ESXI_ROOT_PASSWORD")
		if esxiRootPassword == "" {
			log.Warn("ESXI_ROOT_PASSWORD not provided. A password prompt will need to be filled.")
			return ssh.ExecSSH(allArgs...)
		} else {
			stdin := fmt.Sprintf("%s\n", esxiRootPassword)
			return ssh.ExecSSHStdin(stdin, allArgs...)
		}
	},
}
