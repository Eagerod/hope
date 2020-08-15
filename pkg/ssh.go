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

type ExecSSHFunc func(args ...string) error
type GetSSHFunc func(args ...string) (string, error)
type ExecSCPFunc func(args ...string) error

var ExecSSH ExecSSHFunc = func(args ...string) error {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	fmt.Println("ssh", args)

	osCmd := exec.Command("ssh", args...)
	osCmd.Stdin = os.Stdin
    osCmd.Stdout = os.Stdout
    osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var GetSSH GetSSHFunc = func(args ...string) (string, error) {
	// For now, this is just implemented by using commands, but in the end,
	//   it may be fun to try out using golang.org/x/crypto/ssh
	fmt.Println("ssh", args)

	osCmd := exec.Command("ssh", args...)
	output, err := osCmd.CombinedOutput()
	return string(output), err
}

var execSCP ExecSCPFunc = func(args ...string) error {
	osCmd := exec.Command("scp", args...)
	osCmd.Stdin = os.Stdin
    osCmd.Stdout = os.Stdout
    osCmd.Stderr = os.Stderr

	return osCmd.Run()
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

			if strings.HasPrefix(privateKey, "~") {
				home, err := homedir.Dir()
				if err != nil {
					return err
				}

				privateKey = strings.Replace(privateKey, "~", home, 1)
			}
			publicKey := fmt.Sprintf("%s.pub", privateKey)

			if _, err = os.Stat(publicKey); err != nil && os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Failed to find public key to set up authorized_keys from %s", privateKey))
			}

			destination := fmt.Sprintf("%s:tmp.pub", ip)
			err = execSCP(publicKey, destination)

			if err != nil {
				return err
			}

			// TODO: cat the new public key to the authorized keys, rather than
			//       mangling whatever's already there.
			// TODO: Find the user's home directory and put the .ssh directory
			//       in the right place.
			c := fmt.Sprintf("'mkdir -p .ssh && mv tmp.pub .ssh/authorized_keys && chmod 700 .ssh && chmod 600 .ssh/authorized_keys'")
			err = ExecSSH(ip, "sh", "-c", c)
			if err != nil {
				return err
			}

			// https://unix.stackexchange.com/a/36687/258222
			return ExecSSH(ip, "restorecon", "-R", "-v", "~/.ssh")
		}
	}

	return err
}

func TestPasswordlessSudo(ip string) error {
	return ExecSSH(ip, "sudo", "sh -c 'exit'")
}

func SetupPasswordlessSudo(ip string) error {
	return errors.New("This hasn't been implemented yet.")
}
