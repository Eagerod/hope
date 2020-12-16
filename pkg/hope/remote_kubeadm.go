package hope

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

	if err := ssh.ExecSSH(node.ConnectionString(), "kubeadm", "reset"); err != nil {
		return err
	}

	if nodeName != "" && kubectl != nil {
		if err := kubeutil.ExecKubectl(kubectl, "delete", "node", nodeName); err != nil {
			return err
		}
	}

	return nil
}
