package hope

import (
	"fmt"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

// Just forwards to `SetLoadBalancerHosts`.
// There may be a time where this does more.
func InitLoadBalancer(log *logrus.Entry, node *Node, masters *[]Node) error {
	log.Debug("Starting to bootstrap a simple NGINX load balancer for API Servers at ", node.Host)
	return SetLoadBalancerHosts(log, node, masters)
}

func SetLoadBalancerHosts(log *logrus.Entry, node *Node, masters *[]Node) error {
	if len(*masters) == 0 {
		log.Warn("Setting empty load balancer hosts.")
	}

	masterUpstreamContents := ""
	for _, master := range *masters {
		masterUpstreamContents = fmt.Sprintf("%s\nserver %s:6443;", masterUpstreamContents, master.Host)
	}

	connectionString := node.ConnectionString()

	if err := ssh.ExecSSH(connectionString, "mkdir", "-p", "/etc/nginx"); err != nil {
		return err
	}

	dest := fmt.Sprintf("%s:/etc/nginx/nginx.conf", connectionString)
	populatedConfig := fmt.Sprintf(NginxConfig, masterUpstreamContents)
	if err := scp.ExecSCPBytes([]byte(populatedConfig), dest); err != nil {
		return err
	}

	err := ssh.ExecSSH(connectionString, "sh", "-c", "'docker kill $(docker ps -f expose=6443 -q)'")
	if err != nil {
		log.Info("Failed to kill existing nginx containers on load balancer")
		log.Info(err)
	}

	return ssh.ExecSSH(
		connectionString,
		"docker", "run", "-d",
		"-v", "/etc/nginx/nginx.conf:/etc/nginx/nginx.conf",
		"-p", "6443:6443",
		"--restart", "unless-stopped",
		"nginx:1.19.4",
	)
}
