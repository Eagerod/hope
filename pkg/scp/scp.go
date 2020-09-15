package scp

import (
	"io/ioutil"
	"os"
	"os/exec"
)

type ExecSCPFunc func(args ...string) error
type ExecSCPBytesFunc func(bytes []byte, dest string) error

var ExecSCP ExecSCPFunc = func(args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("scp", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var ExecSCPBytes ExecSCPBytesFunc = func(bytes []byte, dest string) error {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}

	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.Write(bytes); err != nil {
		return err
	}

	if err = tmpfile.Close(); err != nil {
		return err
	}

	if err = ExecSCP(tmpfile.Name(), dest); err != nil {
		return err
	}

	return nil
}
