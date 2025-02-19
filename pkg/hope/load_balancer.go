package hope

import (
	"fmt"
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
func InitLoadBalancer(log *logrus.Entry, node *Node, masters *[]Node) error {
	log.Debug("Starting to bootstrap a simple NGINX load balancer for API Servers at ", node.Host)
	return SetLoadBalancerHosts(log, node, masters)
}

func SetLoadBalancerHosts(log *logrus.Entry, node *Node, masters *[]Node) error {
	if len(*masters) == 0 {
		log.Warn("Setting empty load balancer hosts.")
	}

	connectionString := node.ConnectionString()

	// In the case where there are no masters yet, send traffic to a black
	//   hole.
	// Prevents Nginx from crash looping; upstream servers need at least one
	//   endpoint.
	masterUpstreamContents := ""
	if len(*masters) == 0 {
		masterUpstreamContents = "server 0.0.0.0:6443;"
	} else {
		masterIps := []string{}
		for _, master := range *masters {
			masterUpstreamContents = fmt.Sprintf("%s\n        server %s:6443;", masterUpstreamContents, master.Host)
			masterIps = append(masterIps, fmt.Sprintf("%s:6443", master.Host))
		}
		log.Infof("Setting load balancer upstreams to: %s", strings.Join(masterIps, ", "))
	}
	populatedConfig := fmt.Sprintf(NginxConfig, masterUpstreamContents)
	configTempFilename := uuid.New().String()
	dest := fmt.Sprintf("%s:%s", connectionString, configTempFilename)
	if err := scp.ExecSCPBytes([]byte(populatedConfig), dest); err != nil {
		return err
	}

	runningContainer, err := ssh.GetSSH(connectionString, "sudo", "docker", "ps", "-f", "expose=6443", "-q")
	if err != nil {
		return err
	}

	runningContainer = strings.TrimSpace(runningContainer)

	// TODO: Parameterize nginx version?
	// If a container is already running, just update its config.
	// If not, create the initial config + create the container.
	var statements []string
	if runningContainer == "" {
		statements = []string{
			"mkdir -p /etc/nginx",
			fmt.Sprintf("mv %s /etc/nginx/nginx.conf", configTempFilename),
			"chown root:root /etc/nginx/nginx.conf",
			"docker run -d -v /etc/nginx/nginx.conf:/etc/nginx/nginx.conf -p 6443:6443 --restart unless-stopped nginx:1.19.4",
		}
	} else {
		// Volume needs to keep the same inode, so have to trunc
		//   and append.
		statements = []string{
			fmt.Sprintf("cat %s > /etc/nginx/nginx.conf", configTempFilename),
			fmt.Sprintf("docker exec -i %s nginx -s reload", runningContainer),
			fmt.Sprintf("rm %s", configTempFilename),
		}
	}

	script := fmt.Sprintf("'%s'", strings.Join(statements, ";\n"))
	return ssh.ExecSSH(connectionString, "sudo", "sh", "-ec", script)
}
