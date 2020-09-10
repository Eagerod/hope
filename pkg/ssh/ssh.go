package ssh

import (
	"os"
	"os/exec"
)

type ExecSSHFunc func(args ...string) error

var ExecSSH ExecSSHFunc = func(args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}
