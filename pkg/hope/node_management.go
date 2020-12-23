package hope

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

func setupCommonNodeRequirements(log *logrus.Entry, node *Node) error {
	if !node.IsKubernetesNode() {
		return fmt.Errorf("Node has role %s, should not prepare as Kubernetes node", node.Role)
	}

	if err := TestCanSSHWithoutPassword(node); err != nil {
		return err
	}

	log.Debug("Preparing Kubernetes components at ", node.Host)

	connectionString := node.ConnectionString()

	// TODO: Create a function in ssh pkg that allows for running
	//   multi-statement commands on the target without needing to manually
	//   construct the string.
	// TODO: Consider writing these files using file provisioners in Packer
	//   instead?
	commands := []string{
		"mkdir -p /etc/sysconfig",
		"echo \"\" > /etc/sysconfig/docker-storage",
		"echo \"\" > /etc/sysconfig/docker-storage-setup",
		fmt.Sprintf("echo \"%s\" > /etc/docker/daemon.json", DockerDaemonJson),
		fmt.Sprintf("echo \"%s\" > /etc/sysctl.d/k8s.conf", K8SConf),
		fmt.Sprintf("echo \"%s\" > /proc/sys/net/ipv4/ip_forward", IpForward),
	}
	commandString := fmt.Sprintf("'%s'", strings.Join(commands, " && "))

	if err := ssh.ExecSSH(connectionString, "sudo", "sh", "-c", commandString); err != nil {
		return err
	}

	// Various other setups.
	if err := ssh.ExecSSH(connectionString, "sudo", "sed", "-i", "'/--exec-opt native.cgroupdriver/d'", "/usr/lib/systemd/system/docker.service"); err != nil {
		return err
	}

	if err := ssh.ExecSSH(connectionString, "sudo", "sysctl", "-p"); err != nil {
		return err
	}

	if err := DisableSwapOnRemote(node); err != nil {
		return err
	}

	// TODO: Create a function in ssh pkg that allows for running
	//   multi-statement commands on the target without needing to manually
	//   construct the string.
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
	if err := ssh.ExecSSH(connectionString, "sudo", "bash", "-c", daemonsScript); err != nil {
		return err
	}

	if err := DisableSelinuxOnRemote(node); err != nil {
		return err
	}

	return nil
}

func CreateClusterMaster(log *logrus.Entry, node *Node, podNetworkCidr string) error {
	if !node.IsMaster() {
		return fmt.Errorf("Node has role %s and should not be set up as a Kubernetes master", node.Role)
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
		return fmt.Errorf("Node has role %s and should not be set up as a Kubernetes node", node.Role)
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

	joinComponents := strings.Split(joinCommand, " ")
	allArguments := append([]string{node.ConnectionString()}, joinComponents...)
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
	if err := ssh.ExecSSH(connectionString, "sudo", "hostnamectl", "set-hostname", hostname); err != nil {
		return err
	}

	// TODO: _Might_ be worth dropping word boundaries on the sed script?
	log.Debug("Replacing all instances of ", existingHostname, " in /etc/hosts")
	sedScript := fmt.Sprintf("'s/%s/%s/g'", existingHostname, hostname)
	if err := ssh.ExecSSH(connectionString, "sudo", "sed", "-i", sedScript, "/etc/hosts"); err != nil {
		return err
	}

	// Host _should_ come up before SSH times out.
	log.Info("Restarting networking on ", node.Host)
	if err := ssh.ExecSSH(connectionString, "sudo", "systemctl", "restart", "network"); err != nil {
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
		return fmt.Errorf("Node init aborted. Hostname not confirmed (%s != %s)", trimmedHostname, trimmedInput)
	}

	return nil
}
