package hope

import (
	"fmt"
	"regexp"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

func KubeadmResetRemote(log *logrus.Entry, kubectl *kubeutil.Kubectl, node *Node, force bool) error {
	log.Debug("Searching for node name for host: ", node.Host)

	nodeName := ""
	if kubectl != nil {
		nodeName, err := kubeutil.NodeNameFromHost(kubectl, node.Host)
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
	}

	args := []string{
		node.ConnectionString(),
		"sudo",
		"kubeadm",
		"reset",
	}
	if force {
		args = append(args, "--force")
	}

	if err := ssh.ExecSSH(args...); err != nil {
		return err
	}

	if nodeName != "" && kubectl != nil {
		if err := kubeutil.ExecKubectl(kubectl, "delete", "node", nodeName); err != nil {
			return err
		}
	}

	return nil
}

func KubeadmGetClusterCertificateKey(log *logrus.Entry, node *Node) (string, error) {
	output, err := ssh.GetSSH(node.ConnectionString(), "sudo", "kubeadm", "init", "phase", "upload-certs", "--upload-certs")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		match, err := regexp.MatchString("[0-9a-f]{64}", line)
		if err != nil {
			return "", err
		}

		if match {
			return line, nil
		}
	}

	return "", fmt.Errorf("Failed to find cert key from existing master node: %s", node.Host)
}
