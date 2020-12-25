package hope

import (
	"fmt"
	"path"
	"strings"
)

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

// Just forwards to `SetLoadBalancerHosts`.
// There may be a time where this does more.
func InitLoadBalancer(log *logrus.Entry, node *Node) error {
	log.Debug("Starting to bootstrap a simple NGINX load balancer for API Servers at ", node.Host)
	return SetLoadBalancerHosts(log, node, &[]Node{})
}

func SetLoadBalancerHosts(log *logrus.Entry, node *Node, masters *[]Node) error {
	if len(*masters) == 0 {
		log.Warn("Setting empty load balancer hosts.")
	}

	connectionString := node.ConnectionString()

	// In the case where there are no masters yet, send traffic to a black
	//   hole.
	masterUpstreamContents := ""
	if len(*masters) == 0 {
		masterUpstreamContents = "server 0.0.0.0:6443;"
	} else {
		for _, master := range *masters {
			masterUpstreamContents = fmt.Sprintf("%s\nserver %s:6443;", masterUpstreamContents, master.Host)
		}
	}
	populatedConfig := fmt.Sprintf(NginxConfig, masterUpstreamContents)

	// Because this string ends up being an escaping nightmare when attempting
	//   to write it out directly in a set of statements, copy the file into
	//   the authenticated user's home directory, then copy with root to where
	//   nginx wants it.
	// Pretty sketchy building up the path in the way it is.
	configTempFilename := uuid.New().String()
	dest := fmt.Sprintf("%s:%s", connectionString, configTempFilename)
	if err := scp.ExecSCPBytes([]byte(populatedConfig), dest); err != nil {
		return err
	}

	output, err := ssh.GetSSH(connectionString, "pwd")
	if err != nil {
		return err
	}

	configTempPath := path.Join(strings.TrimSpace(output), configTempFilename)

	// TODO: Parameterize nginx version?
	statements := []string{
		"mkdir -p /etc/nginx",
		fmt.Sprintf("cp %s /etc/nginx/nginx.conf", configTempPath),
		"chown root:root /etc/nginx/nginx.conf",
		fmt.Sprintf("rm -f %s", configTempPath),
		"docker kill $(docker ps -f expose=6443 -q) || true",
		"docker run -d -v /etc/nginx/nginx.conf:/etc/nginx/nginx.conf -p 6443:6443 --restart unless-stopped nginx:1.19.4",
	}

	script := fmt.Sprintf("'%s'", strings.Join(statements, ";\n"))
	return ssh.ExecSSH(node.ConnectionString(), "sudo", "sh", "-ec", script)
}
