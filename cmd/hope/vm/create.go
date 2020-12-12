package vm

import (
	"encoding/json"
	"errors"
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
	"github.com/Eagerod/hope/pkg/fileutil"
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a VM on the specified host.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		vmName := args[1]

		vms, err := getVMs()
		if err != nil {
			return err
		}

		vm, err := vmSpec(vmName)
		if err != nil {
			return err
		}

		// These next ~30 lines were basically copy-pasted from image.go
		// Find the output path the vm image should have been written to
		//   locally, and bail if it doesn't exist yet.
		// There may be more validation that can be done, but it's not really
		//   needed.
		vmDir := path.Join(vms.RootDir, vm.Name)
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
			if err := replaceParametersInDirectory(tempDir, vm.Parameters); err != nil {
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

		packerOutDir := packerSpec.Builders[0].OutputDirectory
		if stat, err := os.Stat(packerOutDir); err != nil {
			if os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Failed to find VM output directory at %s", packerOutDir))
			}
		} else {
			if !stat.IsDir() {
				return errors.New(fmt.Sprintf("VM output is not a directory at %s", packerOutDir))
			}
		}

		// Copy to remote.
		// TODO: Optimize with something like rsync/just checking hashes
		//   before copying, depending on how much I want to add more
		//   dependencies.
		remoteVmfsPath := path.Join("/vmfs/volumes/Main/ovfs", packerSpec.Builders[0].VMName)
		remoteVMPath := fmt.Sprintf("%s:%s", hostname, remoteVmfsPath)
		if err := scp.ExecSCP("-r", packerOutDir, remoteVMPath); err != nil {
			return err
		}

		// Exec OVF tool to start VM.
		// https://www.virtuallyghetto.com/2012/05/how-to-deploy-ovfova-in-esxi-shell.html
		// Note: Right now, this requires manual intervention, since it
		//   doesn't provide a username and password to the invocation.
		// Might be worth introducing some kind of a utility to let private
		//   arguments still get passed without them printing out, or setting
		//   up ExecSSH to have a version that accepts stdin.
		remoteOvfPath := path.Join(remoteVmfsPath, packerSpec.Builders[0].VMName + ".ovf")
		return ssh.ExecSSH(hostname,
			"/vmfs/volumes/Main/bin/ovftool/ovftool",
			"--diskMode=thin",
			"--datastore=Main",
			"--net:'VM Network=VM Network'",
			remoteOvfPath,
			"vi://localhost",
		)
	},
}
