package hope

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

func KubeadmResetRemote(log *logrus.Entry, host string, force bool) error {
	// URL parsing is a bit better at identifying parameters if there's a
	//   protocol on the string passed in, so fake in ssh as the protocol to
	//   help it parse a little more reliably.
	host_url, err := url.Parse(fmt.Sprintf("ssh://%s", host))
	if err != nil {
		return err
	}

	log.Debug("Searching for node name for host: ", host_url.Host)

	nodesOutput, err := kubeutil.GetKubectl("get", "nodes", "-o", "custom-columns=NODE:metadata.name,IP:status.addresses[?(@.type=='InternalIP')].address")
	if err != nil {
		log.Error(nodesOutput)
		return err
	}

	// First row is headers
	nodeRows := strings.Split(nodesOutput, "\n")[1:]

	// Search within each row for whatever second column contains the IP
	//   address we're looking for.
	// TODO: May actually have to check both prefix and suffix in case a
	//   hostname shows up here.
	var nodeName string = ""
	for _, nodeRow := range nodeRows {
		if strings.HasSuffix(nodeRow, host_url.Host) {
			nodeName = strings.Split(nodeRow, " ")[0]
		}
	}

	if nodeName == "" && !force {
		return errors.New(fmt.Sprintf("Failed to find a node with IP: %s", host_url.Host))
	} else if force {
		log.Info("Did not find node in the cluster.")
	} else {
		log.Info("Draining node ", nodeName, " from the cluster")

		if err := kubeutil.ExecKubectl("drain", nodeName, "--ignore-daemonsets"); err != nil {
			return err
		}
	}

	err = ssh.ExecSSH(host, "kubeadm", "reset")
	if err != nil {
		return err
	}

	if nodeName != "" {
		if err := kubeutil.ExecKubectl("delete", "node", nodeName); err != nil {
			return err
		}
	}

	return nil
}
