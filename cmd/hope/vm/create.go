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

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/ssh"
)

var createCmdVmName string
var createCmdMemory string
var createCmdCpu string

func initCreateCmdFlags() {
	createCmd.Flags().StringVarP(&createCmdVmName, "name", "n", "", "what to name the virtual machine. If left blank, defaults to the hope file name.")
	createCmd.Flags().StringVarP(&createCmdMemory, "memory", "m", "", "how much memory to given the created VM.")
	createCmd.Flags().StringVarP(&createCmdCpu, "cpu", "c", "", "how many vCPUs to given the created VM.")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		if createCmdVmName == "" {
			return errors.New("Must provide a VM name")
		}

		hypervisor, err := utils.GetNode(hypervisorName)
		if err != nil {
			return err
		}

		if !hypervisor.IsHypervisor() {
			return fmt.Errorf("Node %s is not a hypervisor node; cannot create a VM on it", hypervisorName)
		}

		vms, err := utils.GetVMs()
		if err != nil {
			return err
		}

		vm, err := utils.VMSpec(vmName)
		if err != nil {
			return err
		}

		vmDir := path.Join(vms.Root, vm.Name)

		log.Debug(fmt.Sprintf("Copying contents of %s for parameter replacement.", vmDir))
		tempDir, err := utils.ReplaceParametersInDirectoryCopy(vmDir, vm.Parameters)
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

		datastoreRoot := path.Join("/", "vmfs", "volumes", hypervisor.Datastore)
		vmOvfName := fmt.Sprintf("%s.ovf", packerSpec.Builders[0].VMName)
		remoteOvfPath := path.Join(datastoreRoot, "ovfs", packerSpec.Builders[0].VMName, vmOvfName)
		allArgs := []string{
			hypervisor.ConnectionString(),
			path.Join(datastoreRoot, "bin", "ovftool", "ovftool"),
			"--diskMode=thin",
			fmt.Sprintf("--datastore=%s", hypervisor.Datastore),
			fmt.Sprintf("--name=%s", createCmdVmName),
			fmt.Sprintf("--net:'%s=%s'", sourceNetworkName, hypervisor.Network),
		}

		if createCmdCpu != "" {
			cpuArg := fmt.Sprintf("--numberOfCpus:'*'=%s", createCmdCpu)
			allArgs = append(allArgs, cpuArg)
		}

		if createCmdMemory != "" {
			memoryArg := fmt.Sprintf("--memorySize:'*'=%s", createCmdMemory)
			allArgs = append(allArgs, memoryArg)
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
