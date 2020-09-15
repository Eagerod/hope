package hope

import (
	"fmt"
	"net/url"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

func KubeadmResetRemote(log *logrus.Entry, kubectl *kubeutil.Kubectl, host string, force bool) error {
	// URL parsing is a bit better at identifying parameters if there's a
	//   protocol on the string passed in, so fake in ssh as the protocol to
	//   help it parse a little more reliably.
	host_url, err := url.Parse(fmt.Sprintf("ssh://%s", host))
	if err != nil {
		return err
	}

	log.Debug("Searching for node name for host: ", host_url.Host)

	nodeName, err := kubeutil.NodeNameFromHost(host_url.Host)
	if err != nil && !force {
		return err
	} else if force {
		log.Info("Did not find node in the cluster.")
	} else {
		log.Info("Draining node ", nodeName, " from the cluster")

		if err := kubeutil.ExecKubectl(kubectl, "drain", nodeName, "--ignore-daemonsets"); err != nil {
			return err
		}
	}

	err = ssh.ExecSSH(host, "kubeadm", "reset")
	if err != nil {
		return err
	}

	if nodeName != "" {
		if err := kubeutil.ExecKubectl(kubectl, "delete", "node", nodeName); err != nil {
			return err
		}
	}

	return nil
}
