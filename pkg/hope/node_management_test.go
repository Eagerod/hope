package hope

import (
	"testing"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetupCommonNodeRequirementsNotKubernetesNode(t *testing.T) {
	node := Node{
		Role: "load-balancer",
	}
	err := setupCommonNodeRequirements(log.WithFields(log.Fields{}), &node)
	assert.Error(t, err, "Node has role load-balancer, should not prepare as Kubernetes node")
}
