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

func InitLoadBalancer(log *logrus.Entry, host string) error {
	log.Debug("Starting to bootstrap a simple NGINX load balancer for API Servers at ", host)

	if err := ssh.ExecSSH(host, "mkdir", "-p", "/etc/nginx"); err != nil {
		return err
	}

	dest := fmt.Sprintf("%s:/etc/nginx/nginx.conf", host)
	if err := scp.ExecSCPBytes([]byte(NginxConfig), dest); err != nil {
		return err
	}

	err := ssh.ExecSSH(host, "sh", "-c", "'docker kill $(docker ps -f expose=6443 -q)'")
	if err != nil {
		log.Info("Failed to kill existing nginx containers on load balancer")
		log.Info(err)
	}

	return ssh.ExecSSH(host, "docker", "run", "-d", "-v", "/etc/nginx/nginx.conf:/etc/nginx/nginx.conf", "-p", "6443:6443", "nginx:1.19.4")
}
