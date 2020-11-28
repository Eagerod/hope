package vm

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

type VMImageSpec struct {
	Name       string
	Parameters []string
}

type VMs struct {
	Images  []VMImageSpec
	RootDir string `mapstructure:"root_dir"`
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

// Copied from parent package.
// Need to figure out the canonical way of doing this.
// Import parent package?
// Define in pkg?
func replaceParametersInFile(path string, parameters []string) (string, error) {
	t, err := hope.TextSubstitutorFromFilepath(path)
	if err != nil {
		return "", err
	}

	return replaceParametersWithSubstitutor(t, parameters)
}

func replaceParametersWithSubstitutor(t *hope.TextSubstitutor, parameters []string) (string, error) {
	envParams := []string{}
	directParams := map[string]string{}
	for _, value := range parameters {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 1 {
			envParams = append(envParams, value)
		} else {
			directParams[parts[0]] = parts[1]
		}
	}

	if err := t.SubstituteTextFromEnv(envParams); err != nil {
		return "", err
	}

	if err := t.SubstituteTextFromMap(directParams); err != nil {
		return "", err
	}

	return string(*t.Bytes), nil
}
