package vm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/fileutil"
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

		// These next ~30 lines were basically copy-pasted from image.go
		// Find the output path the vm image should have been written to
		//   locally, and bail if it doesn't exist yet.
		// There may be more validation that can be done, but it's not really
		//   needed.
		// And they should all go into pkg.
		vmDir := path.Join(vms.Root, vm.Name)
		tempDir, err := ioutil.TempDir("", "*")
		if err != nil {
			return err
		}

		defer os.RemoveAll(tempDir)

		log.Debug(fmt.Sprintf("Copying contents of %s to %s for contents replacement.", vmDir, tempDir))
		if err := fileutil.CopyDirectory(vmDir, tempDir); err != nil {
			return err
		}

		if len(vm.Parameters) != 0 {
			if err := utils.ReplaceParametersInDirectory(tempDir, vm.Parameters); err != nil {
				return err
			}
		}

		tempPackerJsonPath := path.Join(tempDir, "packer.json")
		bytes, err := ioutil.ReadFile(tempPackerJsonPath)
		if err != nil {
			return err
		}

		var packerSpec PackerSpec
		if err := json.Unmarshal(bytes, &packerSpec); err != nil {
			return err
		}

		// Exec OVF tool to start VM.
		// https://www.virtuallyghetto.com/2012/05/how-to-deploy-ovfova-in-esxi-shell.html
		// Note: Right now, this requires manual intervention, since it
		//   doesn't provide a username and password to the invocation.
		// Might be worth introducing some kind of a utility to let private
		//   arguments still get passed without them printing out, or setting
		//   up ExecSSH to have a version that accepts stdin.
		vmOvfName := fmt.Sprintf("%s.ovf", packerSpec.Builders[0].VMName)
		remoteOvfPath := path.Join("/", "vmfs", "volumes", hypervisor.Datastore, "ovfs", packerSpec.Builders[0].VMName, vmOvfName)
		remoteDatastoreName := fmt.Sprintf("--datastore=%s", hypervisor.Datastore)
		ovfToolPath := path.Join("/", "vmfs", "volumes", hypervisor.Datastore, "bin", "ovftool", "ovftool")
		allArgs := []string{
			hypervisor.ConnectionString(),
			ovfToolPath,
			"--diskMode=thin",
			remoteDatastoreName,
			"--net:'VM Network=VM Network'",
		}

		if createCmdCpu != "" {
			cpuArg := fmt.Sprintf("--numberOfCpus:'*'=%s", createCmdCpu)
			allArgs = append(allArgs, cpuArg)
		}

		if createCmdMemory != "" {
			memoryArg := fmt.Sprintf("--memorySize:'*'=%s", createCmdMemory)
			allArgs = append(allArgs, memoryArg)
		}

		if createCmdVmName != "" {
			nameArg := fmt.Sprintf("--name=%s", createCmdVmName)
			allArgs = append(allArgs, nameArg)
		}

		allArgs = append(allArgs, remoteOvfPath, "vi://root@localhost")

		return ssh.ExecSSH(allArgs...)
	},
}
