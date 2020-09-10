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
	host_url, err := url.Parse(host)
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
