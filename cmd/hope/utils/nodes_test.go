package utils

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/hope/hypervisors"
)

var testNodes []hope.Node = []hope.Node{
	{
		Name:      "beast1",
		Role:      hope.NodeRoleHypervisor.String(),
		Engine:    "esxi",
		Host:      "192.168.10.40",
		User:      "root",
		Datastore: "Main",
		Network:   "VM Network",
	},
	{
		Name:       "test-load-balancer",
		Role:       hope.NodeRoleLoadBalancer.String(),
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-01",
		Role:       hope.NodeRoleMaster.String(),
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-02",
		Role:       hope.NodeRoleMaster.String(),
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-master-03",
		Role:       hope.NodeRoleMaster.String(),
		Hypervisor: "beast1",
		User:       "packer",
	},
	{
		Name:       "test-node-01",
		Role:       hope.NodeRoleNode.String(),
		Hypervisor: "beast1",
		User:       "packer",
	},
}

var oldToHypervisor func(hope.Node) (hypervisors.Hypervisor, error) = toHypervisor

type MockHypervisor struct {
	node hope.Node
}

func (m *MockHypervisor) ListNodes() ([]string, error) {
	nodes := []string{}
	for _, n := range testNodes {
		if !n.IsHypervisor() {
			nodes = append(nodes, n.Name)
		}
	}
	return nodes, nil
}

func (m *MockHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	node.Hypervisor = ""
	node.Host = node.Name
	return node, nil
}

func (m *MockHypervisor) UnderlyingNode() (hope.Node, error) {
	return m.node, nil
}

func toHypervisorStub(node hope.Node) (hypervisors.Hypervisor, error) {
	if !node.IsHypervisor() {
		return nil, fmt.Errorf("Not a hypervisor")
	}
	return &MockHypervisor{node}, nil
}

// Implemented as a suite to allow manipulating the hypervisor factory
//   function.
type NodesTestSuite struct {
	suite.Suite
}

func (s *NodesTestSuite) SetupTest() {
	toHypervisor = toHypervisorStub
}

func (s *NodesTestSuite) TeardownTest() {
	toHypervisor = oldToHypervisor
}

// Actual test method to run the suite
func TestNodes(t *testing.T) {
	suite.Run(t, new(NodesTestSuite))
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func (s *NodesTestSuite) TestGetNodes() {
	t := s.T()
	resetViper(t)

	nodes, err := getNodes()
	assert.Nil(t, err)
	assert.Equal(t, testNodes, nodes)
}

func (s *NodesTestSuite) TestGetNode() {
	t := s.T()
	resetViper(t)

	expected := testNodes[5]
	expected.Host = "test-node-01"
	expected.Hypervisor = ""

	node, err := GetNode("test-node-01")
	assert.Nil(t, err)

	assert.Equal(t, expected, node)

	node, err = GetNode("sets-node-01")
	assert.Equal(t, "Failed to find a node named sets-node-01", err.Error())
}

func (s *NodesTestSuite) TestHasNode() {
	t := s.T()
	resetViper(t)

	assert.True(t, HasNode("test-node-01"))
	assert.False(t, HasNode("sets-node-01"))
}

func (s *NodesTestSuite) TestGetAnyMaster() {
	t := s.T()
	resetViper(t)

	expected := testNodes[2]
	expected.Host = "test-master-01"
	expected.Hypervisor = ""

	node, err := GetAnyMaster()
	assert.Nil(t, err)

	assert.Equal(t, node, expected)
}

func (s *NodesTestSuite) TestGetHypervisors() {
	t := s.T()
	resetViper(t)

	hypervisors, err := GetHypervisors()
	assert.Nil(t, err)

	assert.Equal(t, 1, len(hypervisors))

	node, err := hypervisors[0].UnderlyingNode()
	assert.Equal(t, testNodes[0], node)
}

func (s *NodesTestSuite) TestGetHypervisor() {
	t := s.T()
	resetViper(t)

	expected := testNodes[0]

	hypervisor, err := GetHypervisor("beast1")
	assert.Nil(t, err)

	n, err := hypervisor.UnderlyingNode()
	assert.Nil(t, err)
	assert.Equal(t, expected, n)

	hypervisor, err = GetHypervisor("test-node-01")
	assert.Nil(t, hypervisor)
	assert.Equal(t, "Not a hypervisor", err.Error())

	hypervisor, err = GetHypervisor("sets-node-01")
	assert.Nil(t, hypervisor)
	assert.Equal(t, "Failed to find a hypervisor named sets-node-01", err.Error())
}

func (s *NodesTestSuite) TestGetAvailableMasters() {
	t := s.T()
	resetViper(t)

	expectedOrig := testNodes[2:5]
	expected := []hope.Node{}
	for i, n := range expectedOrig {
		n.Host = fmt.Sprintf("test-master-0%d", i+1)
		n.Hypervisor = ""
		expected = append(expected, n)
	}

	masters, err := GetAvailableMasters()
	assert.Nil(t, err)

	assert.Equal(t, expected, masters)
}

func (s *NodesTestSuite) TestGetLoadBalancer() {
	t := s.T()
	resetViper(t)

	expected := testNodes[1]
	expected.Host = "test-load-balancer"
	expected.Hypervisor = ""

	node, err := GetLoadBalancer()
	assert.Nil(t, err)

	assert.Equal(t, node, expected)
}
