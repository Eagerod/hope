package hope

import (
	"fmt"
	"net/url"
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
func InitLoadBalancer(log *logrus.Entry, host string, masterHosts []string) error {
	log.Debug("Starting to bootstrap a simple NGINX load balancer for API Servers at ", host)
	return SetLoadBalancerHosts(log, host, masterHosts)
}

func SetLoadBalancerHosts(log *logrus.Entry, loadBalancerHost string, masterHosts []string) error {
	if len(masterHosts) == 0 {
		log.Warn("Setting empty load balancer hosts.")
	}

	masterUpstreamContents := ""
	for _, master := range masterHosts {
		hostUrl, err := url.Parse(fmt.Sprintf("ssh://%s", master))
		if err != nil {
			return err
		}

		masterUpstreamContents = fmt.Sprintf("%s\nserver %s:6443;", masterUpstreamContents, hostUrl.Host)
	}

	if err := ssh.ExecSSH(loadBalancerHost, "mkdir", "-p", "/etc/nginx"); err != nil {
		return err
	}

	dest := fmt.Sprintf("%s:/etc/nginx/nginx.conf", loadBalancerHost)
	populatedConfig := fmt.Sprintf(NginxConfig, masterUpstreamContents)
	if err := scp.ExecSCPBytes([]byte(populatedConfig), dest); err != nil {
		return err
	}

	err := ssh.ExecSSH(loadBalancerHost, "sh", "-c", "'docker kill $(docker ps -f expose=6443 -q)'")
	if err != nil {
		log.Info("Failed to kill existing nginx containers on load balancer")
		log.Info(err)
	}

	return ssh.ExecSSH(
		loadBalancerHost,
		"docker", "run", "-d",
		"-v", "/etc/nginx/nginx.conf:/etc/nginx/nginx.conf",
		"-p", "6443:6443",
		"--restart", "unless-stopped",
		"nginx:1.19.4",
	)
}
