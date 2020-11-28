package vm

import (
	"errors"
	"fmt"
)

import (
	"github.com/spf13/viper"
)

type VMImageSpec struct {
	Name string
	Parameters []string
}

type VMs struct {
	Images []VMImageSpec
	RootDir string  `mapstructure:"root_dir"`
}

func getVMs() (*VMs, error) {
	var vms VMs
	err := viper.UnmarshalKey("vms", &vms)

	nameMap := map[string]bool{}
	for _, vm := range vms.Images {
		if _, ok := nameMap[vm.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Multiple VMs found in configuration file named: %s", vm.Name))
		}
		nameMap[vm.Name] = true
	}

	return &vms, err
}
