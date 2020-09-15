package hope

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

func setupCommonNodeRequirements(log *logrus.Entry, masterIp string) error {
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

	log.Debug("Preparing Kubernetes components at ", masterIp)

	// Write all the empty files that should exist first.
	dest := fmt.Sprintf("%s:%s", masterIp, "/etc/sysconfig/docker-storage")
	if err := scp.ExecSCPBytes([]byte(""), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/sysconfig/docker-storage-setup")
	if err := scp.ExecSCPBytes([]byte(""), dest); err != nil {
		return err
	}

	// Write files with contents.
	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/docker/daemon.json")
	if err := scp.ExecSCPBytes([]byte(DockerDaemonJson), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/etc/sysctl.d/k8s.conf")
	if err := scp.ExecSCPBytes([]byte(K8SConf), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", masterIp, "/proc/sys/net/ipv4/ip_forward")
	if err := scp.ExecSCPBytes([]byte(IpForward), dest); err != nil {
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

	return nil
}

func CreateClusterMaster(log *logrus.Entry, masterIp string, podNetworkCidr string) error {
	if err := setupCommonNodeRequirements(log, masterIp); err != nil {
		return err
	}

	podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
	if err := ssh.ExecSSH(masterIp, "kubeadm", "init", podNetworkCidrArg); err != nil {
		return err
	}

	return nil
}

func CreateClusterNode(log *logrus.Entry, nodeIp string, masterIp string) error {
	if err := setupCommonNodeRequirements(log, nodeIp); err != nil {
		return err
	}

	hostname, err := ssh.GetSSH(nodeIp, "hostname")
	if err != nil {
		return err
	}

	trimmedHostname := strings.TrimSpace(hostname)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Creating a node in the cluster called:", trimmedHostname)
	fmt.Print("If this is correct, re-enter the hostname: ")

	inputHostname, _ := reader.ReadString('\n')
	trimmedInput := strings.TrimSpace(inputHostname)

	if trimmedHostname != trimmedInput {
		return errors.New(fmt.Sprintf("Node init aborted. Hostname not confirmed (%s != %s).", trimmedHostname, trimmedInput))
	}

	joinCommand, err := ssh.GetSSH(masterIp, "kubeadm", "token", "create", "--print-join-command")
	if err != nil {
		return err
	}

	joinComponents := strings.Split(joinCommand, " ")
	allArguments := append([]string{nodeIp}, joinComponents...)
	if err := ssh.ExecSSH(allArguments...); err != nil {
		return err
	}

	return nil
}

func TaintNodeByHost(kubectl *kubeutil.Kubectl, host string, taint string) error {
	nodeName, err := kubeutil.NodeNameFromHost(kubectl, host)
	if err != nil {
		return err
	}

	if err := kubeutil.ExecKubectl(kubectl, "taint", "nodes", nodeName, taint); err != nil {
		return err
 	}

 	return nil
}

func SetHostname(log *logrus.Entry, host string, hostname string, force bool) error {
	existingHostname, err := ssh.GetSSH(host, "hostname")
	if err != nil {
		return nil
	}
	existingHostname = strings.TrimSpace(existingHostname)

	if !force {
		log.Trace("Testing hostname on ", host, " before committing any changes...")

		if hostname == existingHostname {
			log.Debug("Hostname of ", host, " is already ", hostname, ". Skipping hostname setting.")

			return nil
		} else {
			log.Trace("Hostname of ", host, " is ", existingHostname)
		}
	}

	log.Trace("Setting hostname to ", hostname)
	if err := ssh.ExecSSH(host, "hostnamectl", "set-hostname", hostname); err != nil {
		return err
	}

	// TODO: _Might_ be worth dropping word boundaries on the sed script?
	log.Debug("Replacing all instances of ", existingHostname, " in /etc/hosts")
	sedScript := fmt.Sprintf("'s/%s/%s/g'", existingHostname, hostname)
	if err := ssh.ExecSSH(host, "sed", "-i", sedScript, "/etc/hosts"); err != nil {
		return err
	}

	// Host _should_ come up before SSH times out.
	log.Info("Restarting networking on ", host)
	if err := ssh.ExecSSH(host, "systemctl", "restart", "network"); err != nil {
	}

	return nil
}
