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

func setupCommonNodeRequirements(log *logrus.Entry, node *Node) error {
	if !node.IsRoleValid() {
		return errors.New(fmt.Sprintf("Node has role %s, should not prepare as Kubernetes node.", node.Role))
	}

	if err := TestCanSSHWithoutPassword(node); err != nil {
		return err
	}

	log.Debug("Preparing Kubernetes components at ", node.Host)

	connectionString := node.ConnectionString()
	// Write all the empty files that should exist first.
	dest := fmt.Sprintf("%s:%s", connectionString, "/etc/sysconfig/docker-storage")
	if err := scp.ExecSCPBytes([]byte(""), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", connectionString, "/etc/sysconfig/docker-storage-setup")
	if err := scp.ExecSCPBytes([]byte(""), dest); err != nil {
		return err
	}

	// Write files with contents.
	dest = fmt.Sprintf("%s:%s", connectionString, "/etc/docker/daemon.json")
	if err := scp.ExecSCPBytes([]byte(DockerDaemonJson), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", connectionString, "/etc/sysctl.d/k8s.conf")
	if err := scp.ExecSCPBytes([]byte(K8SConf), dest); err != nil {
		return err
	}

	dest = fmt.Sprintf("%s:%s", connectionString, "/proc/sys/net/ipv4/ip_forward")
	if err := scp.ExecSCPBytes([]byte(IpForward), dest); err != nil {
		return err
	}

	// Various other setups.
	if err := ssh.ExecSSH(connectionString, "sed", "-i", "'/--exec-opt native.cgroupdriver/d'", "/usr/lib/systemd/system/docker.service"); err != nil {
		return err
	}

	ssh.ExecSSH(connectionString, "sed", "-i", "'s/--log-driver=journald//'", "/etc/sysconfig/docker")

	if err := ssh.ExecSSH(connectionString, "sysctl", "-p"); err != nil {
		return err
	}

	if err := DisableSwapOnRemote(node); err != nil {
		return err
	}

	daemonsScript := fmt.Sprintf("\"%s\"", strings.Join(
		[]string{
			"systemctl daemon-reload",
			"systemctl enable docker",
			"systemctl enable kubelet",
			"systemctl start docker",
			"systemctl start kubelet",
		},
		" && ",
	))
	if err := ssh.ExecSSH(connectionString, "bash", "-c", daemonsScript); err != nil {
		return err
	}

	if err := DisableSelinuxOnRemote(node); err != nil {
		return err
	}

	return nil
}

func CreateClusterMaster(log *logrus.Entry, node *Node, podNetworkCidr string) error {
	if !node.IsMaster() {
		return errors.New(fmt.Sprintf("Node has role %s and should not be set up as a Kubernetes master", node.Role))
	}

	if err := setupCommonNodeRequirements(log, node); err != nil {
		return err
	}

	if err := forceUserToEnterHostnameToContinue(node); err != nil {
		return err
	}

	connectionString := node.ConnectionString()
	podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
	if err := ssh.ExecSSH(connectionString, "kubeadm", "init", podNetworkCidrArg); err != nil {
		return err
	}

	return nil
}

func CreateClusterNode(log *logrus.Entry, node *Node, masterIp string) error {
	if !node.IsNode() {
		return errors.New(fmt.Sprintf("Node has role %s and should not be set up as a Kubernetes node", node.Role))
	}

	if err := setupCommonNodeRequirements(log, node); err != nil {
		return err
	}

	if err := forceUserToEnterHostnameToContinue(node); err != nil {
		return err
	}

	joinCommand, err := ssh.GetSSH(masterIp, "kubeadm", "token", "create", "--print-join-command")
	if err != nil {
		return err
	}

	connectionString := node.ConnectionString()
	joinComponents := strings.Split(joinCommand, " ")
	allArguments := append([]string{connectionString}, joinComponents...)
	if err := ssh.ExecSSH(allArguments...); err != nil {
		return err
	}

	return nil
}

func TaintNodeByHost(kubectl *kubeutil.Kubectl, node *Node, taint string) error {
	nodeName, err := kubeutil.NodeNameFromHost(kubectl, node.Host)
	if err != nil {
		return err
	}

	if err := kubeutil.ExecKubectl(kubectl, "taint", "nodes", nodeName, taint); err != nil {
		return err
	}

	return nil
}

func SetHostname(log *logrus.Entry, node *Node, hostname string, force bool) error {
	connectionString := node.ConnectionString()

	existingHostname, err := ssh.GetSSH(connectionString, "hostname")
	if err != nil {
		return nil
	}
	existingHostname = strings.TrimSpace(existingHostname)

	if !force {
		log.Trace("Testing hostname on ", node.Host, " before committing any changes...")

		if hostname == existingHostname {
			log.Debug("Hostname of ", node.Host, " is already ", hostname, ". Skipping hostname setting.")

			return nil
		} else {
			log.Trace("Hostname of ", node.Host, " is ", existingHostname)
		}
	}

	log.Trace("Setting hostname to ", hostname)
	if err := ssh.ExecSSH(connectionString, "hostnamectl", "set-hostname", hostname); err != nil {
		return err
	}

	// TODO: _Might_ be worth dropping word boundaries on the sed script?
	log.Debug("Replacing all instances of ", existingHostname, " in /etc/hosts")
	sedScript := fmt.Sprintf("'s/%s/%s/g'", existingHostname, hostname)
	if err := ssh.ExecSSH(connectionString, "sed", "-i", sedScript, "/etc/hosts"); err != nil {
		return err
	}

	// Host _should_ come up before SSH times out.
	log.Info("Restarting networking on ", node.Host)
	if err := ssh.ExecSSH(connectionString, "systemctl", "restart", "network"); err != nil {
	}

	return nil
}

func forceUserToEnterHostnameToContinue(node *Node) error {
	connectionString := node.ConnectionString()

	hostname, err := ssh.GetSSH(connectionString, "hostname")
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

	return nil
}
