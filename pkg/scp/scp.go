package scp

import (
	"os"
	"os/exec"
)

type ExecSCPFunc func(args ...string) error

var ExecSCP ExecSCPFunc = func(args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("scp", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}
