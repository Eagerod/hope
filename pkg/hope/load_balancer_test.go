package hope

import (
	"testing"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoadBalancerConfigurationFileNoMasters(t *testing.T) {
	masters := []Node{}
	config := loadBalancerConfigurationFile(log.WithFields(log.Fields{}), &masters)
	assert.Contains(t, config, "0.0.0.0:6443")
}

func TestLoadBalancerConfigurationFileMasters(t *testing.T) {
	masters := []Node{
		Node { Host: "192.168.1.254" },
	}
	config := loadBalancerConfigurationFile(log.WithFields(log.Fields{}), &masters)
	assert.Contains(t, config, "192.168.1.254:6443")
}
