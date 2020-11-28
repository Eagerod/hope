package packer

import (
	"os"
	"os/exec"
)

type ExecPackerFunc func(args ...string) error
type ExecPackerWdFunc func(workDir string, args ...string) error

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
