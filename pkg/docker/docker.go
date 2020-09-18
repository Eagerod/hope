package docker

import (
	"os"
	"os/exec"
)

// TODO: Add a bunch of methods that wrap individual commands.
// Invocations require the package name anyways, so something like:
//   docker.Pull(...)
// is a lot nicer than
//   docker.ExecDocker("pull", ...)
type ExecDockerFunc func(args ...string) error

var ExecDocker ExecDockerFunc = func(args ...string) error {
	osCmd := exec.Command("docker", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}
