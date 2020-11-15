package hope

import (
	"bufio"
	"errors"
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
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

func setupCommonNodeRequirements(log *logrus.Entry, masterIp string) error {
	if err := TestCanSSHWithoutPassword(masterIp); err != nil {
		return err
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
	if err := ssh.ExecSSH(masterIp, "bash", "-c", daemonsScript); err != nil {
		return err
	}

	if err := DisableSelinuxOnRemote(masterIp); err != nil {
		return err
	}

	return nil
}

func CreateClusterMaster(log *logrus.Entry, masterIp string, podNetworkCidr string, loadBalancerHost string, allMasters []string) error {
	if err := setupCommonNodeRequirements(log, masterIp); err != nil {
		return err
	}

	if err := forceUserToEnterHostnameToContinue(masterIp); err != nil {
		return err
	}

	// Search through the existing masters to see if this node is being added
	//   as a master to an existing control plane, or if this will be the
	//   first master in the pool.
	for _, aMaster := range allMasters {
		if aMaster == masterIp {
			continue
		}

		remoteAdminConfPath := "/etc/kubernetes/admin.conf"
		grepRegexp := "'\\s+server: https://api\\.internal\\.aleemhaji\\.com:6443'"

		if err := ssh.ExecSSH(aMaster, "grep", "-E", grepRegexp, "-q", remoteAdminConfPath); err != nil {
			log.Warn("Other master node", aMaster, "isn't connected to load balancer.")
			continue
		}

		// From this master node, pull a control plane certificate key, and
		//   use it to run the kubeadm join command.
		output, err := ssh.GetSSH(aMaster, "kubeadm", "init", "phase", "upload-certs", "--upload-certs")
		if err != nil {
			return err
		}

		certKey := ""
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			match, err := regexp.MatchString("[0-9a-f]{64}", line)
			if err != nil {
				return err
			}

			if match {
				certKey = line
				break
			}
		}

		if certKey == "" {
			return errors.New(fmt.Sprintf("Failed to find cert key from selected master node: %s", aMaster))
		}

		joinCommand, err := ssh.GetSSH(aMaster, "kubeadm", "token", "create", "--print-join-command")
		if err != nil {
			return err
		}

		joinComponents := strings.Split(strings.TrimSpace(joinCommand), " ")
		allArguments := append([]string{masterIp}, joinComponents...)
		allArguments = append(allArguments, "--control-plane", "--certificate-key", certKey)

		if err := ssh.ExecSSH(allArguments...); err != nil {
			return err
		}

		return nil
	}

	// At this point, it's very probable that the node being added is the
	//   first master node to be created in the cluster.
	// Note: While this is running, the load balancer has to be updated to
	//   include the master host.
	// Maybe wait 20 seconds before updating the lb.
	podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
	allArgs := []string{masterIp, "kubeadm", "init", podNetworkCidrArg}
	if loadBalancerHost != "" {
		loadBalancerEndpoint := fmt.Sprintf("%s:%s", loadBalancerHost, "6443")
		allArgs = append(allArgs, "--control-plane-endpoint", loadBalancerEndpoint, "--upload-certs")
	}

	if err := ssh.ExecSSH(allArgs...); err != nil {
		return err
	}

	return nil
}

func CreateClusterNode(log *logrus.Entry, nodeIp string, masterIp string) error {
	if err := setupCommonNodeRequirements(log, nodeIp); err != nil {
		return err
	}

	if err := forceUserToEnterHostnameToContinue(nodeIp); err != nil {
		return err
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

func forceUserToEnterHostnameToContinue(host string) error {
	hostname, err := ssh.GetSSH(host, "hostname")
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
