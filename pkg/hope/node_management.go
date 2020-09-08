package hope

import (
	"fmt"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func CreateClusterMaster(log *logrus.Entry, masterIp string, podNetworkCidr string) error {
	log.Debug("Running some tests to ensure this process can be run properly...")

	if err := ssh.TestCanSSH(masterIp); err != nil {
		// Try to recover this.
		if err = ssh.TryConfigureSSH(masterIp); err != nil {
			return err
		}

		log.Info("Configured passwordless SSH using the identity file that SSH uses for this connection by default")
	} else {
		log.Trace("Passwordless SSH has already been configured on ", masterIp)
	}

	log.Debug("Creating cluster at", masterIp)

	// Write all the empty files that should exist first.
	dest := fmt.Sprintf("%s:%s", masterIp, "/etc/sysconfig/docker-storage")
	if err := ssh.CopyStringToDest("", dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/sysconfig/docker-storage-setup")
	if err := ssh.CopyStringToDest("", dest); err != nil {
		return err
	}

	// Write files with contents.
	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/docker/daemon.json")
	if err := ssh.CopyStringToDest(DockerDaemonJson, dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/sysctl.d/k8s.conf")
	if err := ssh.CopyStringToDest(K8SConf, dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/proc/sys/net/ipv4/ip_forward")
	if err := ssh.CopyStringToDest(IpForward, dest); err != nil {
		return err
	}

	// Various other setups.
	if err := ssh.ExecSSH(masterIp, "sed", "-i", "'/--exec-opt native.cgroupdriver/d'", "/usr/lib/systemd/system/docker.service"); err != nil {
		return err
	}

	ssh.ExecSSH(masterIp, "sed", "-i", "'s/--log-driver=journald//'", "/etc/sysconfig/docker")

	if err := ssh.ExecSSH(masterIp, "sysctl", "-p"); err != nil {
		return err
	}

	if err := DisableSwapOnRemote(masterIp); err != nil {
		return err
	}

	if err := ssh.ExecSSH(masterIp, "systemctl", "daemon-reload"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(masterIp, "systemctl", "enable", "docker"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(masterIp, "systemctl", "enable", "kubelet"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(masterIp, "systemctl", "start", "docker"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(masterIp, "systemctl", "enable", "kubelet"); err != nil {
		return err
	}

	if err := DisableSelinuxOnRemote(masterIp); err != nil {
		return err
	}

	podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
	if err := ssh.ExecSSH(masterIp, "kubeadm", "init", podNetworkCidrArg); err != nil {
		return err
	}

	return nil
}
