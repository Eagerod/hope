package packer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

type ExecPackerFunc func(args ...string) error
type ExecPackerWdFunc func(workDir string, args ...string) error
type ExecPackerWdEnvFunc func(workDir string, env *map[string]string, args ...string) error

// JsonBuilder - A few pieces of the Packer JSON builders list.
type JsonBuilder struct {
	VMName          string `json:"vm_name"`
	OutputDirectory string `json:"output_directory"`
	VMXData			map[string]string `json:"vmx_data"`
}

// JsonSpec - Wrapper for the builders list.
type JsonSpec struct {
	Builders []JsonBuilder
}

var ExecPacker ExecPackerFunc = func(args ...string) error {
	osCmd := exec.Command("packer", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var ExecPackerWd ExecPackerWdFunc = func(workDir string, args ...string) error {
	osCmd := exec.Command("packer", args...)
	osCmd.Dir = workDir
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var ExecPackerWdEnv ExecPackerWdEnvFunc = func(workDir string, env *map[string]string, args ...string) error {
	osCmd := exec.Command("packer", args...)
	osCmd.Dir = workDir
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	osCmd.Env = os.Environ()
	for key, value := range *env {
		osCmd.Env = append(osCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	return osCmd.Run()
}

func SpecFromPath(path string) (*JsonSpec, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var spec JsonSpec
	if err := json.Unmarshal(bytes, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}
