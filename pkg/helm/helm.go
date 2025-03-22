package helm

import (
	"os"
	"os/exec"
)

type ExecHelmFunc func(args ...string) error
type GetHelmFunc func(args ...string) (string, error)

var ExecHelm ExecHelmFunc = func(args ...string) error {
	osCmd := exec.Command("helm", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var GetHelm GetHelmFunc = func(args ...string) (string, error) {
	osCmd := exec.Command("helm", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}
