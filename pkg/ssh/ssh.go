package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

import (
	homedir "github.com/mitchellh/go-homedir"
)

import (
	"github.com/Eagerod/hope/pkg/scp"
)

type ExecSSHFunc func(args ...string) error
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

var GetSSH GetSSHFunc = func(args ...string) (string, error) {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

// Attempt to SSH into a machine without allowing password authentication.
func TestCanSSH(ip string) error {
	return ExecSSH("-o", "PasswordAuthentication=no", "-o", "BatchMode=yes", ip, "exit")
}

// See what SSH key this host is trying to use, and try copying it over to the
//   appropriate place using password auth.
func TryConfigureSSH(ip string) error {
	output, err := GetSSH("-G", ip)

	for _, s := range strings.Split(output, "\n") {
		if strings.HasPrefix(s, "identityfile") {
			fmt.Fprintln(os.Stderr, "Attempting to configure SSH on the remote machine")
			fmt.Fprintln(os.Stderr, "You will be asked for the password for", ip, "several times")

			privateKey := strings.Replace(s, "identityfile ", "", 1)
			privateKey, err = homedir.Expand(privateKey)
			if err != nil {
				return err
			}

			publicKey := fmt.Sprintf("%s.pub", privateKey)

			if _, err = os.Stat(publicKey); err != nil && os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Failed to find public key to set up authorized_keys from %s", privateKey))
			}

			destination := fmt.Sprintf("%s:tmp.pub", ip)

			if err := scp.ExecSCP(publicKey, destination); err != nil {
				return err
			}

			// TODO: cat the new public key to the authorized keys, rather than
			//       mangling whatever's already there.
			// TODO: Find the user's home directory and put the .ssh directory
			//       in the right place.
			c := fmt.Sprintf("'mkdir -p .ssh && mv tmp.pub .ssh/authorized_keys && chmod 700 .ssh && chmod 600 .ssh/authorized_keys'")
			if err = ExecSSH(ip, "sh", "-c", c); err != nil {
				return err
			}

			// https://unix.stackexchange.com/a/36687/258222
			return ExecSSH(ip, "restorecon", "-R", "-v", "~/.ssh")
		}
	}

	return err
}
