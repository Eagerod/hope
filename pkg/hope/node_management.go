package hope

import (
	"bufio"
	"fmt"
	"os"
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

// Sets up any configuration on Kubernetes nodes that are common between
// control-plane nodes, and worker nodes.
// TODO: Consider writing these files using file provisioners in Packer
// instead?
func setupCommonNodeRequirements(log *logrus.Entry, node *Node) error {
	if !node.IsKubernetesNode() {
		return fmt.Errorf("Node has role %s, should not prepare as Kubernetes node", node.Role)
	}

	if err := TestCanSSHWithoutPassword(node); err != nil {
		return err
	}

	log.Debug("Preparing Kubernetes components at ", node.Host)

	connectionString := node.ConnectionString()

	// Different versions of kubeabm will install their kubeadm.conf under
	//   different paths; take the first one found from known paths.
	possibleKubeadmConfigPaths := []string{
		"/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf",
		"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
	}

	commandStr := ""
	for _, p := range possibleKubeadmConfigPaths {
		commandStr = fmt.Sprintf("%s if [ -f \"%s\" ]; then echo \"%s\"; exit; fi;", commandStr, p, p)
	}

	kubeadmConfigPath, err := ssh.GetSSH(connectionString, commandStr)
	if err != nil {
		return err
	}

	kubeadmConfigPath = strings.TrimSpace(kubeadmConfigPath)

	// Make sure the Kubelet cgroups driver is also systemd.
	if err := ssh.ExecSSH(connectionString, "sudo", "sed", "-i", "'s#Environment=\"KUBELET_CONFIG_ARGS=.*#Environment=\"KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml --cgroup-driver=systemd\"#g'", kubeadmConfigPath); err != nil {
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
			"systemctl enable kubelet",
			"systemctl restart kubelet",
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

func CreateClusterMaster(log *logrus.Entry, node *Node, podNetworkCidr string, loadBalancer *Node, loadBalancerHost string, masters *[]Node, force bool) error {
	if !node.IsMaster() {
		return fmt.Errorf("Node has role %s and should not be set up as a Kubernetes master", node.Role)
	}

	if err := setupCommonNodeRequirements(log, node); err != nil {
		return err
	}

	if !force {
		if err := forceUserToEnterHostnameToContinue(node); err != nil {
			return err
		}
	}

	if loadBalancer == nil {
		return createClusterMasterStandalone(log, node, podNetworkCidr)
	}

	// Search through the existing masters to see if this node is being added
	//   as a master to an existing control plane, or if this will be the
	//   first master in the pool.
	connectionString := node.ConnectionString()
	loadBalancerEndpoint := fmt.Sprintf("%s:%s", loadBalancerHost, "6443")

	// Loop over the list of defined masters, and filter it down to a list
	//   only includes masters that have already been initialized and added to
	//   the load balancer, plus the one about to added.
	lbMasters := []Node{*node}
	grepRegexp := fmt.Sprintf("'\\s+server: https://%s'", regexp.QuoteMeta(loadBalancerEndpoint))
	for _, aMaster := range *masters {
		aMasterCs := aMaster.ConnectionString()
		if aMasterCs == connectionString {
			continue
		}

		if err := ssh.ExecSSH(aMasterCs, "sudo", "grep", "-E", grepRegexp, "-q", "/etc/kubernetes/admin.conf"); err != nil {
			log.Infof("Other master node %s isn't connected to load balancer.", aMaster.Host)
			continue
		}

		lbMasters = append(lbMasters, aMaster)
	}

	if err := SetLoadBalancerHosts(log, loadBalancer, &lbMasters); err != nil {
		return err
	}

	// If no other defined masters existed, or no other masters were
	//   configured to use the defined load balancer, set up this node as the
	//   first node in the load balanced group.
	if len(lbMasters) == 1 {
		podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
		allArgs := []string{connectionString, "sudo", "kubeadm", "init", podNetworkCidrArg}
		allArgs = append(allArgs, "--control-plane-endpoint", loadBalancerEndpoint, "--upload-certs")

		return ssh.ExecSSH(allArgs...)
	}

	// This master is being added to an existing pool.
	otherMaster := lbMasters[1]
	certKey, err := KubeadmGetClusterCertificateKey(log, &otherMaster)
	if err != nil {
		return err
	}

	existingMasters := lbMasters[1:]
	joinCommand, err := KubeadmGetClusterJoinCommandFromAnyMaster(&existingMasters)
	if err != nil {
		return err
	}

	joinComponents := strings.Split(strings.TrimSpace(joinCommand), " ")
	allArguments := append([]string{connectionString}, "sudo")
	allArguments = append(allArguments, joinComponents...)
	allArguments = append(allArguments, "--control-plane", "--certificate-key", certKey)

	return ssh.ExecSSH(allArguments...)
}

func createClusterMasterStandalone(log *logrus.Entry, node *Node, podNetworkCidr string) error {
	podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
	return ssh.ExecSSH(node.ConnectionString(), "sudo", "kubeadm", "init", podNetworkCidrArg)
}

func CreateClusterNode(log *logrus.Entry, node *Node, masters *[]Node, force bool) error {
	if !node.IsNode() {
		return fmt.Errorf("Node has role %s and should not be set up as a Kubernetes node", node.Role)
	}

	if err := setupCommonNodeRequirements(log, node); err != nil {
		return err
	}

	if !force {
		if err := forceUserToEnterHostnameToContinue(node); err != nil {
			return err
		}
	}

	joinCommand, err := KubeadmGetClusterJoinCommandFromAnyMaster(masters)
	if err != nil {
		return err
	}

	joinComponents := strings.Split(joinCommand, " ")
	allArguments := append([]string{node.ConnectionString(), "sudo"}, joinComponents...)
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
		}

		log.Trace("Hostname of ", node.Host, " is ", existingHostname)
	}

	// Before setting the hostname, make sure the primary interface is set to
	//   bring itself back up on a networking restart.
	// If it's not, the device may not turn back on with the network.
	// TODO: Test on different distros with different ways of managing the
	//   network.
	ethInterface, err := ssh.GetSSH(connectionString, "sudo", "sh", "-c", "'ip route get 8.8.8.8 | head -1 | awk \"{print \\$5}\"'")
	if err != nil {
		return err
	}

	ethInterface = strings.TrimSpace(ethInterface)
	ethScript := fmt.Sprintf("auto %s\nallow-hotplug %s\niface %s inet dhcp\n", ethInterface, ethInterface, ethInterface)

	scripts := []string{
		fmt.Sprintf("sed -i \"/%s/d\" /etc/network/interfaces", ethInterface),
		fmt.Sprintf("printf \"%s\" >> /etc/network/interfaces", ethScript),
	}

	combinedScript := fmt.Sprintf("'%s'", strings.Join(scripts, " && "))
	if err := ssh.ExecSSH(connectionString, "sudo", "sh", "-c", combinedScript); err != nil {
		return err
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
	script := "'if [ -f /etc/init.d/networking ]; then /etc/init.d/networking restart; else systemctl restart network; fi'"
	ssh.ExecSSH(connectionString, "sudo", "sh", "-c", script)

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
