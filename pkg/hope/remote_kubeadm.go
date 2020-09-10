package hope

import (
	"errors"
	"fmt"
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
	components := strings.Split(host, "@")
	if len(components) != 2 {
		return errors.New(fmt.Sprintf("Do no understand host: %s", host))
	}

	// Try to run as many intermediate operations as possible regardless of the
	//   force flag.
	nodeHost := components[1]

	log.Debug("Searching for node name for host: ", nodeHost)

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
		if strings.HasSuffix(nodeRow, nodeHost) {
			nodeName = strings.Split(nodeRow, " ")[0]
		}
	}

	if nodeName == "" && !force {
		return errors.New(fmt.Sprintf("Failed to find a node with IP: %s", nodeHost))
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
