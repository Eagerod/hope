package hope

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

import (
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func DisableSwapOnRemote(node *Node) error {
	connectionString := node.ConnectionString()

	// TODO: Execute in a single SSH session.
	if err := ssh.ExecSSH(connectionString, "sed", "-i", "'/ swap / s/^/#/'", "/etc/fstab"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(connectionString, "swapoff", "-a"); err != nil {
		return err
	}

	return nil
}

func DisableSelinuxOnRemote(node *Node) error {
	connectionString := node.ConnectionString()

	// TODO: Execute in a single SSH session.
	// If this is running on a non-SELinux distro, just bail without trying to
	//   do anything meaningful.
	if err := ssh.ExecSSH(connectionString, "which", "getenforce"); err != nil {
		return nil
	}

	enforcing, err := ssh.GetSSH(connectionString, "getenforce")
	if err != nil {
		return err
	}

	if strings.TrimSpace(enforcing) != "Disabled" {
		if err := ssh.ExecSSH(connectionString, "setenforce", "0"); err != nil {
			return err
		}
	}

	if err := ssh.ExecSSH(connectionString, "sed", "-i", "'s/SELINUX=enforcing/SELINUX=disabled/g'", "/etc/selinux/config"); err != nil {
		return err
	}

	return nil
}

func EnsureSSHWithoutPassword(log *logrus.Entry, node *Node) error {
	connectionString := node.ConnectionString()

	if err := TestCanSSHWithoutPassword(node); err == nil {
		log.Trace("Passwordless SSH has already been configured for ", connectionString)
		return nil
	}

	// Before trying to set up passwordless SSH on the machine, see if we can
	//   even SSH into the machine by password.
	// It's possible the machine has already been configured allow only pubkey
	//   auth, and this can't proceed at all.
	// This invocation is pretty well guaranteed to fail; don't check its
	//   returned error.
	out, _ := ssh.GetErrorSSH("-v", "-o", "Batchmode=yes", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", connectionString)

	// Find a line that says "Authentications that can continue" and
	//   password, or keyboard-interactive.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "debug1: Authentications that can continue") {
			if strings.Contains(line, "password") || strings.Contains(line, "keyboard-interactive") {
				log.Debug("Password authentication may be possible on ", connectionString, ". Attempting password session")
				if err := TryConfigureSSH(log, node); err != nil {
					return err
				} else {
					return nil
				}
			}
		}
	}

	return errors.New("Failed to set up passwordless SSH because SSH key not present on remote, and password auth is disabled.")
}

// Attempt to SSH into a machine without allowing password authentication.
// Also disables strict host checking to prevent the unattended nature of the
//   execution from causing the script to fail.
func TestCanSSHWithoutPassword(node *Node) error {
	return ssh.ExecSSH("-o", "Batchmode=yes", "-o", "StrictHostKeyChecking=no", "-o", "PasswordAuthentication=no", node.ConnectionString(), "exit")
}

// See what SSH key this host is trying to use, and try copying it over to the
//   appropriate place using password auth.
func TryConfigureSSH(log *logrus.Entry, node *Node) error {
	connectionString := node.ConnectionString()
	output, err := ssh.GetSSH("-G", connectionString)

	for _, s := range strings.Split(output, "\n") {
		if strings.HasPrefix(s, "identityfile") {
			// Print direct to console, because loglevel shouldn't
			//   prevent this from showing up.
			fmt.Fprintln(os.Stderr, "Attempting to configure SSH on the remote machine")
			fmt.Fprintln(os.Stderr, "You will be asked for the password for", connectionString, "several times")

			privateKey := strings.Replace(s, "identityfile ", "", 1)
			privateKey, err = homedir.Expand(privateKey)
			if err != nil {
				return err
			}

			publicKey := fmt.Sprintf("%s.pub", privateKey)

			if err := CopySSHKeyToAuthorizedKeys(log, publicKey, node); err != nil {
				return err
			}

			// https://unix.stackexchange.com/a/36687/258222
			return ssh.ExecSSH(connectionString, "sh", "-c", "'type restorecon && restorecon -R -v ~/.ssh || echo >&2 \"Failed to run restorecon\"'")
		}
	}

	return err
}

func CopySSHKeyToAuthorizedKeys(log *logrus.Entry, keyPath string, node *Node) error {
	if _, err := os.Stat(keyPath); err != nil && os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("Failed to find public key to set up authorized_keys from %s", keyPath))
	}

	connectionString := node.ConnectionString()

	// TODO: Maybe confirm that this actually is a public key, and if it looks
	//   like a private key, try to add .pub and see if there's a file there.
	sshPubKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}

	// TODO: This should check to see if the given key already exists in the
	//   authorized keys.
	// TODO: Create a function in ssh pkg that allows for running
	//   multi-statement commands on the target without needing to manually
	//   construct the string.
	commands := []string{
		"mkdir -p $HOME/.ssh",
		"chmod 700 $HOME/.ssh",
		fmt.Sprintf("printf \"%s\n\" >> $HOME/.ssh/authorized_keys", strings.TrimSpace(string(sshPubKey))),
		"chmod 600 $HOME/.ssh/authorized_keys",
	}
	commandString := fmt.Sprintf("'%s'", strings.Join(commands, " && "))

	if err := ssh.ExecSSH(connectionString, "sh", "-c", commandString); err != nil {
		return err
	}

	return nil
}
