package packer

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
)

type ExecPackerFunc func(args ...string) error
type ExecPackerWdFunc func(workDir string, args ...string) error

// JsonBuilder - A few pieces of the Packer JSON builders list.
type JsonBuilder struct {
	VMName          string `json:"vm_name"`
	OutputDirectory string `json:"output_directory"`
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