package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/pkg/fileutil"
	"github.com/Eagerod/hope/pkg/packer"
)

var imageCmdParameterSlice *[]string
var imageCmdDebugPackerFlag bool

func initImageCmdFlags() {
	imageCmdParameterSlice = imageCmd.Flags().StringArrayP("param", "p", []string{}, "parameters to forward to packer's -var")
	imageCmd.Flags().BoolVarP(&imageCmdDebugPackerFlag, "debug-packer", "", false, "pass the debug flag to packer")
}

type PackerBuilder struct {
	VMName string `json:"vm_name"`
	OutputDirectory string `json:"output_directory"`
}

type PackerSpec struct {
	Builders []PackerBuilder
}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Creates a VM image from the defined packer spec.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		vms, err := getVMs()
		if err != nil {
			return err
		}

		var vm *VMImageSpec
		for _, avm := range vms.Images {
			if avm.Name == vmName {
				vm = &avm
				break
			}
		}

		if vm == nil {
			return errors.New(fmt.Sprintf("No VM named %s found in images definitions.", vmName))
		}

		vmDir := path.Join(vms.RootDir, vm.Name)
		log.Trace(fmt.Sprintf("Looking for VM definition in %s", vmDir))

		stat, err := os.Stat(vmDir)
		if err != nil && os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("VM spec directory (%s) not found", vmDir))
		} else if err != nil {
			return err
		}

		if !stat.IsDir() {
			return errors.New(fmt.Sprintf("VM spec directory (%s) is just a file", vmDir))
		}

		packerJsonPath := path.Join(vmDir, "packer.json")
		if _, err := os.Stat(packerJsonPath); err != nil && os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("VM packer file not found", packerJsonPath))
		} else if err != nil {
			return err
		}

		// Copy the directory out to a temporary directory, and iterate
		//   through all the files, running text substitution against them
		//   with the list of given parameters.
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
			// Probably move this to a util/pkg; seems pretty universal.
			err = filepath.Walk(tempDir, func(apath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				str, err := replaceParametersInFile(apath, vm.Parameters)
				if err != nil {
					return err
				}

				return fileutil.WriteFile(str, apath)
			})

			if err != nil {
				return err
			}
		}

		// Check caches to see if I even want to build this again.
		tempPackerJsonPath := path.Join(tempDir, "packer.json")
		bytes, err := ioutil.ReadFile(tempPackerJsonPath)
		if err != nil {
			return err
		}

		var packerSpec PackerSpec
		if err := json.Unmarshal(bytes, &packerSpec); err != nil {
			return err
		}

		packerOutDir := path.Join(packerSpec.Builders[0].OutputDirectory, packerSpec.Builders[0].VMName)
		if stat, err := os.Stat(packerOutDir); err == nil {
			if stat.IsDir() {
				files, err := ioutil.ReadDir("./")
				if err != nil {
					return err
				}

				if len(files) != 0 {
					return errors.New(fmt.Sprintf("Directory at path %s already exists and is not empty.", packerOutDir))
				}
			} else {
				log.Debug(fmt.Sprintf("Will create a new directory at %s...", packerOutDir))
			}
		}

		allArgs := []string{"build"}
		for _, v := range *imageCmdParameterSlice {
			allArgs = append(allArgs, "-var", v)
		}
		if imageCmdDebugPackerFlag {
			allArgs = append(allArgs, "-debug")
		}
		allArgs = append(allArgs, tempPackerJsonPath)

		if os.Getenv("PACKER_CACHE_DIR") == "" {
			log.Warn("PACKER_CACHE_DIR environment variable is not set.")
		}

		log.Info(fmt.Sprintf("Building VM Image: %s", vm.Name))

		return packer.ExecPackerWd(tempDir, allArgs...)
	},
}
