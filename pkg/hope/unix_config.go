package hope

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

import (
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

func DisableSwapOnRemote(remote string) error {
	// TODO: Execute in a single SSH session.
	if err := ssh.ExecSSH(remote, "sed", "-i", "'/ swap / s/^/#/'", "/etc/fstab"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(remote, "swapoff", "-a"); err != nil {
		return err
	}

	return nil
}

func DisableSelinuxOnRemote(remote string) error {
	// TODO: Execute in a single SSH session.
	enforcing, err := ssh.GetSSH(remote, "getenforce")
	if err != nil {
		return err
	}

	if strings.TrimSpace(enforcing) != "Disabled" {
		if err := ssh.ExecSSH(remote, "setenforce", "0"); err != nil {
			return err
		}
	}

	if err := ssh.ExecSSH(remote, "sed", "-i", "'s/SELINUX=enforcing/SELINUX=disabled/g'", "/etc/selinux/config"); err != nil {
		return err
	}

	return nil
}

func EnsureSSHWithoutPassword(log *logrus.Entry, host string) error {
	if err := TestCanSSHWithoutPassword(host); err == nil {
		log.Trace("Passwordless SSH has already been configured on ", host)
		return nil
	}

	// Before trying to set up passwordless SSH on the machine, see if we can
	//   even SSH into the machine by password.
	// It's possible the machine has already been configured allow only pubkey
	//   auth, and this can't proceed at all.
	// This invocation is pretty well guaranteed to fail; don't check its
	//   returned error.
	out, _ := ssh.GetErrorSSH("-v", "-o", "Batchmode=yes", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", host)

	// Find a line that says "Authentications that can continue" and
	//   password.
	// This line existing will mean that password authentication is
	//   enabled on the host.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Authentications that can continue") && strings.Contains(line, "password") {
			log.Debug("Password authentication may be possible on ", host, ". Attempting password session")
			if err := TryConfigureSSH(log, host); err != nil {
				return err
			} else {
				return nil
			}
		}
	}

	return errors.New("Failed to set up passwordless SSH because SSH key not present on remote, and password auth is disabled.")
}

// Attempt to SSH into a machine without allowing password authentication.
// Also disables strict host checking to prevent the unattended nature of the
//   execution from causing the script to fail.
func TestCanSSHWithoutPassword(host string) error {
	return ssh.ExecSSH("-o", "Batchmode=yes", "-o", "StrictHostKeyChecking=no", "-o", "PasswordAuthentication=no", host, "exit")
}

// See what SSH key this host is trying to use, and try copying it over to the
//   appropriate place using password auth.
func TryConfigureSSH(log *logrus.Entry, host string) error {
	output, err := ssh.GetSSH("-G", host)

	for _, s := range strings.Split(output, "\n") {
		if strings.HasPrefix(s, "identityfile") {
			// Print direct to console, because loglevel shouldn't
			//   prevent this from showing up.
			fmt.Fprintln(os.Stderr, "Attempting to configure SSH on the remote machine")
			fmt.Fprintln(os.Stderr, "You will be asked for the password for", host, "several times")

			privateKey := strings.Replace(s, "identityfile ", "", 1)
			privateKey, err = homedir.Expand(privateKey)
			if err != nil {
				return err
			}

			publicKey := fmt.Sprintf("%s.pub", privateKey)

			if err := CopySSHKeyToAuthorizedKeys(log, publicKey, host); err != nil {
				return err
			}

			// https://unix.stackexchange.com/a/36687/258222
			return ssh.ExecSSH(host, "restorecon", "-R", "-v", "~/.ssh")
		}
	}

	return err
}

func CopySSHKeyToAuthorizedKeys(log *logrus.Entry, keyPath string, host string) error {
	if _, err := os.Stat(keyPath); err != nil && os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("Failed to find public key to set up authorized_keys from %s", keyPath))
	}

	// TODO: Because this is run as separate ssh sessions, each session
	//   results in asking the user to password for the destination host.
	//   Limiting this to a single invocation would be nice.
	// TODO: Don't even copy the public key to a file on the remote.
	//   Just write it through stdin from the ssh command.
	destination := fmt.Sprintf("%s:tmp.pub", host)
	if err := scp.ExecSCP(keyPath, destination); err != nil {
		return err
	}

	if err := ssh.ExecSSH(host, "sh", "-c", "'mkdir -p $HOME/.ssh && chmod 700 $HOME/.ssh'"); err != nil {
		return err
	}

	// TODO: This should check to see if the given key already exists in the
	//   authorized keys.
	if err := ssh.ExecSSH(host, "sh", "-c", "'cat tmp.pub >> $HOME/.ssh/authorized_keys && rm tmp.pub && chmod 600 $HOME/.ssh/authorized_keys'"); err != nil {
		return err
	}

	return nil
}
