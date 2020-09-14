package hope

import (
	"fmt"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func SetHostname(log *logrus.Entry, host string, hostname string, force bool) error {
	existingHostname, err := ssh.GetSSH(host, "hostname")
	if err != nil {
		return nil
	}
	existingHostname = strings.TrimSpace(existingHostname)

	if !force {
		log.Debug("Testing hostname on ", host, " before committing any changes...")

		if hostname == existingHostname {
			log.Info("Hostname of ", host, " is already ", hostname, ". Skipping hostname setting.")

			return nil
		} else {
			log.Debug("Hostname of ", host, " is ", existingHostname)
		}
	}

	if err := ssh.ExecSSH(host, "hostnamectl", "set-hostname", hostname); err != nil {
		return err
	}

	log.Info("Replacing all instances of ", existingHostname, " in /etc/hosts")
	sedScript := fmt.Sprintf("'s/%s/%s/g'", existingHostname, hostname)
	if err := ssh.ExecSSH(host, "sed", "-i", sedScript, "/etc/hosts"); err != nil {
		return err
	}

	log.Info("Restarting networking on ", host)
	if err := ssh.ExecSSH(host, "systemctl", "restart", "network"); err != nil {
		return err
	}

	return nil
}
