package utils

import (
	"fmt"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

func GetVMs() (*hope.VMs, error) {
	var vms hope.VMs
	err := viper.UnmarshalKey("vms", &vms)

	nameMap := map[string]bool{}
	for _, vm := range vms.Images {
		if _, ok := nameMap[vm.Name]; ok {
			return nil, fmt.Errorf("Multiple VMs found in configuration file named: %s", vm.Name)
		}
		nameMap[vm.Name] = true
	}

	return &vms, err
}

func VMSpec(vmName string) (*hope.VMImageSpec, error) {
	vms, err := GetVMs()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms.Images {
		if vm.Name == vmName {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("No VM named %s found in image definitions", vmName)
}
