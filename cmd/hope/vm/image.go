package vm

import (
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
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

var imageCmdParameterSlice *[]string

func initImageCmdFlags() {
	imageCmdParameterSlice = imageCmd.Flags().StringArrayP("param", "p", []string{}, "parameters to forward to packer's -var")
}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Creates a VM image from the defined packer spec.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hypervisorName := args[0]
		vmName := args[1]

		vms, err := utils.GetVMs()
		if err != nil {
			return err
		}

		vm, err := utils.VMSpec(vmName)
		if err != nil {
			return err
		}

		hypervisor, err := utils.GetHypervisor(hypervisorName)
		if err != nil {
			return err
		}

		vmDir := path.Join(vms.Root, vm.Name)
		outputDir := path.Join(vms.Output, vm.Name)
		log.Trace(fmt.Sprintf("Looking for VM definition in %s", vmDir))

		// This is done in advance so that the error can show the user the
		//   real path the file that's expected to load, rather than a path in
		//   the temp directory everything gets copied into.
		packerJsonPath := path.Join(vmDir, "packer.json")
		if _, err := os.Stat(packerJsonPath); err != nil && os.IsNotExist(err) {
			return fmt.Errorf("VM packer file not found at path: %s", packerJsonPath)
		} else if err != nil {
			return err
		}

		// Create full parameter set.
		allParameters := append(vm.Parameters,
			fmt.Sprintf("ESXI_HOST=%s", hypervisor.Host),
			fmt.Sprintf("ESXI_USERNAME=%s", hypervisor.User),
			fmt.Sprintf("ESXI_DATASTORE=%s", hypervisor.Datastore),
			fmt.Sprintf("OUTPUT_DIR=%s", outputDir),
		)

		log.Debug(fmt.Sprintf("Copying contents of %s for parameter replacement.", vmDir))
		tempDir, err := utils.ReplaceParametersInDirectoryCopy(vmDir, allParameters)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		// Check caches to see if I even want to build this again.
		tempPackerJsonPath := path.Join(tempDir, "packer.json")
		packerSpec, err := packer.SpecFromPath(tempPackerJsonPath)
		if err != nil {
			return err
		}

		// Packer runs out of temp dir, so directories have to be absolute.
		packerOutDir := packerSpec.Builders[0].OutputDirectory
		if !path.IsAbs(packerOutDir) {
			return fmt.Errorf("Packer output directory %s must be absolute", packerOutDir)
		}

		if !path.IsAbs(vms.Cache) {
			return fmt.Errorf("Packer cache directory %s must be absolute", vms.Cache)
		}

		if stat, err := os.Stat(packerOutDir); err == nil {
			if stat.IsDir() {
				files, err := ioutil.ReadDir(packerOutDir)
				if err != nil {
					return err
				}

				if len(files) != 0 {
					return fmt.Errorf("Directory at path %s already exists and is not empty", packerOutDir)
				}
			} else {
				log.Debug(fmt.Sprintf("Will create a new directory at %s...", packerOutDir))
			}
		}

		// Try to create a file in the same directory as the output will be.
		// Prevents going through the whole process when the output directory
		//   isn't writable.
		// Seems like a no brainer for packer to do that check.
		if err := os.MkdirAll(packerOutDir, 0755); err != nil {
			return fmt.Errorf("Directory at path %s is not writable; %w", packerOutDir, err)
		}

		allArgs := []string{"build"}
		for _, v := range *imageCmdParameterSlice {
			allArgs = append(allArgs, "-var", v)
		}
		allArgs = append(allArgs, tempPackerJsonPath)

		packerEsxiVncProbeTimeout := os.Getenv("PACKER_ESXI_VNC_PROBE_TIMEOUT")
		if packerEsxiVncProbeTimeout == "" {
			log.Info("PACKER_ESXI_VNC_PROBE_TIMEOUT not set, defaulting to 2s")
			packerEsxiVncProbeTimeout = "2s"
		}

		packerEnvs := map[string]string {
			"PACKER_CACHE_DIR": vms.Cache,
			"PACKER_LOG": "1",
			"PACKER_ESXI_VNC_PROBE_TIMEOUT": packerEsxiVncProbeTimeout,
		}

		log.Info(fmt.Sprintf("Building VM Image: %s", vm.Name))

		if err := packer.ExecPackerWdEnv(tempDir, &packerEnvs, allArgs...); err != nil {
			return err
		}

		// Copy to all hypervisors.
		hypervisors, err := utils.GetHypervisors()
		if err != nil {
			return err
		}

		// Remove the destination file from the Hypervisor before copying,
		//   because SCP is bad at nesting directories, or I'm bad at figuring
		//   out the right arguments.
		for _, hv := range *hypervisors {
			connectionString := hv.ConnectionString()
			scpSrcDir := packerOutDir
			remoteVmfsPath := path.Join("/", "vmfs", "volumes", hv.Datastore, "ovfs", packerSpec.Builders[0].VMName)
			remoteVMPath := fmt.Sprintf("%s:%s", hv.ConnectionString(), remoteVmfsPath)

			if err := ssh.ExecSSH(connectionString, "rm", "-rf", remoteVmfsPath); err != nil {
				return err
			}

			if err := scp.ExecSCP("-pr", scpSrcDir, remoteVMPath); err != nil {
				return err
			}
		}

		return nil
	},
}
