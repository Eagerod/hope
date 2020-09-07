package cmd

import (
	"errors"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
	"github.com/Eagerod/hope/pkg/sliceutil"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap the master node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("This command will attempt to bootstrap a master node.")

		masterIp := args[0]
		podNetworkCidr := viper.GetString("pod_network_cidr")

		if !sliceutil.StringInSlice(masterIp, viper.GetStringSlice("masters")) {
			return errors.New(fmt.Sprintf("Failed to find master %s in config", masterIp))
		}

		fmt.Println("Running some tests to ensure this process can be run properly...")

		if err := ssh.TestCanSSH(masterIp); err != nil {
			// Try to recover this.
			if err = ssh.TryConfigureSSH(masterIp); err != nil {
				return err
			}

			fmt.Println("Configured passwordless SSH using the identity file that SSH uses for this connection by default")
		}

		fmt.Println("Creating cluster at", masterIp)

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

		if err := ssh.ExecSSH(masterIp, "sed", "-i", "'/ swap / s/^/#/'", "/etc/fstab"); err != nil {
			return err
		}

		if err := ssh.ExecSSH(masterIp, "swapoff", "-a"); err != nil {
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

		ssh.ExecSSH(masterIp, "setenforce", "0")

		if err := ssh.ExecSSH(masterIp, "sed", "-i", "'s/SELINUX=enforcing/SELINUX=disabled/g'", "/etc/selinux/config"); err != nil {
			return err
		}

		podNetworkCidrArg := fmt.Sprintf("--pod-network-cidr=%s", podNetworkCidr)
		if err := ssh.ExecSSH(masterIp, "kubeadm", "init", podNetworkCidrArg); err != nil {
			return err
		}

		return nil
	},
}
