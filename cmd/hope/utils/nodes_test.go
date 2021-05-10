package utils

import (
	"testing"

	"github.com/Eagerod/hope/pkg/hope"
	"github.com/stretchr/testify/assert"
)

var testNodes []hope.Node = []hope.Node{
	{
		Name:      "beast1",
		Role:      "hypervisor",
		Host:      "192.168.10.40",
		User:      "root",
		Datastore: "Main",
		Network:   "VM Network",
	},
	{
		Name:       "test-load-balancer",
		Role:       "load-balancer",
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-01",
		Role:       "master",
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-02",
		Role:       "master",
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-03",
		Role:       "master",
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-node-01",
		Role:       "node",
		Hypervisor: "beast1",
		User:       "packer",
	},
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func TestGetNodes(t *testing.T) {
	resetViper(t)

	nodes, err := getNodes()
	assert.Nil(t, err)
	assert.Equal(t, testNodes, *nodes)
}

func TestHasNode(t *testing.T) {
	resetViper(t)

	assert.True(t, HasNode("test-node-01"))
	assert.False(t, HasNode("sets-node-01"))
}

func TestGetHypervisors(t *testing.T) {
	resetViper(t)

	expected := []hope.Node{
		testNodes[0],
	}

	hypervisors, err := GetHypervisors()

	assert.Nil(t, err)
	assert.Equal(t, &expected, hypervisors)
}

func TestGetHypervisor(t *testing.T) {
	resetViper(t)

	expected := testNodes[0]

	hypervisor, err := GetHypervisor("beast1")

	assert.Nil(t, err)
	assert.Equal(t, &expected, hypervisor)

	hypervisor, err = GetHypervisor("test-node-01")

	assert.Nil(t, hypervisor)
	assert.Equal(t, "Node named test-node-01 is not a hypervisor", err.Error())

	hypervisor, err = GetHypervisor("sets-node-01")

	assert.Nil(t, hypervisor)
	assert.Equal(t, "Failed to find a hypervisor named sets-node-01", err.Error())
}
