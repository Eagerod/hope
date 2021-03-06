package ssh

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type ExecSSHFunc func(args ...string) error
type ExecSSHStdinFunc func(stdin string, args ...string) error
type GetSSHFunc func(args ...string) (string, error)

var ExecSSH ExecSSHFunc = func(args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var ExecSSHStdin ExecSSHStdinFunc = func(stdin string, args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = strings.NewReader(stdin)
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var GetSSH GetSSHFunc = func(args ...string) (string, error) {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

var GetErrorSSH GetSSHFunc = func(args ...string) (string, error) {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout

	var stderrBytes bytes.Buffer
	osCmd.Stderr = &stderrBytes
	err := osCmd.Run()

	return stderrBytes.String(), err
}
