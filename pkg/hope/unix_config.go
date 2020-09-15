package hope

import (
	"strings"
)

import (
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
