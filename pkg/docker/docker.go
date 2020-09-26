package docker

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

var UseSudo bool = false

// TODO: Add a bunch of methods that wrap individual commands.
// Invocations require the package name anyways, so something like:
//   docker.Pull(...)
// is a lot nicer than
//   docker.ExecDocker("pull", ...)
type ExecDockerFunc func(args ...string) error

var ExecDocker ExecDockerFunc = func(args ...string) error {
	var osCmd *exec.Cmd
	if UseSudo {
		allArgs := append([]string{"docker"}, args...)
		osCmd = exec.Command("sudo", allArgs...)
	} else {
		osCmd = exec.Command("docker", args...)
	}
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

func SetUseSudo() {
	osCmd := exec.Command("docker", "ps")

	var stderrBytes bytes.Buffer
	osCmd.Stderr = &stderrBytes
	osCmd.Run()

	if strings.Contains(stderrBytes.String(), "permission denied") {
		UseSudo = true
	} else {
		UseSudo = false
	}
}

func AskSudo() error {
	osCmd := exec.Command("sudo", "sh", "-c", "exit")
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}
