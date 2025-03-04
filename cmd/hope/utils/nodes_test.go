package utils

import (
	"fmt"
	"testing"
)

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/hope/hypervisors"
)

func resetViper(t *testing.T) {
	viper.Reset()

	// Assume config file in the project root.
	// Probably bad practice, but better test than having nothing at all.
	viper.AddConfigPath("../../../")
	viper.SetConfigName("hope")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	assert.Nil(t, err)
}

var beast1Node hope.Node = hope.Node{
	Name:      "beast1",
	Role:      hope.NodeRoleHypervisor.String(),
	Engine:    "esxi",
	Host:      "192.168.10.40",
	User:      "root",
	Datastore: "Main",
	Network:   "VM Network",
	Parameters: []string{
		"INSECURE=true",
	},
}
var loadBalancerNode hope.Node = hope.Node{
	Name:       "test-load-balancer",
	Role:       hope.NodeRoleLoadBalancer.String(),
	Hypervisor: "beast1",
	User:       "packer",
	Cpu:        2,
	Memory:     256,
}
var master1Node hope.Node = hope.Node{
	Name:       "test-master-01",
	Role:       hope.NodeRoleMaster.String(),
	Hypervisor: "beast1",
	User:       "packer",
	Cpu:        2,
	Memory:     2048,
}
var master2Node hope.Node = hope.Node{
	Name:   "test-master-02",
	Role:   hope.NodeRoleMaster.String(),
	User:   "packer",
	Host:   "192.168.1.10",
	Cpu:    2,
	Memory: 2048,
}
var master3Node hope.Node = hope.Node{
	Name:       "test-master-03",
	Role:       hope.NodeRoleMaster.String(),
	Hypervisor: "beast1",
	User:       "packer",
	Cpu:        2,
	Memory:     2048,
}
var worker1Node hope.Node = hope.Node{
	Name:       "test-node-01",
	Role:       hope.NodeRoleNode.String(),
	Hypervisor: "beast1",
	User:       "packer",
	Cpu:        2,
	Memory:     4096,
}

var testNodes []hope.Node = []hope.Node{
	beast1Node,
	loadBalancerNode,
	master1Node,
	master2Node,
	master3Node,
	worker1Node,
}

type MockHypervisor struct {
	node hope.Node
}

func (m *MockHypervisor) Initialize(node hope.Node) error {
	return nil
}

func (m *MockHypervisor) ListNodes() ([]string, error) {
	nodes := []string{
		loadBalancerNode.Name,
		master1Node.Name,
		master2Node.Name,
		master3Node.Name,
		worker1Node.Name,
	}
	return nodes, nil
}

func (m *MockHypervisor) ListBuiltImages(vms hope.VMs) ([]string, error) {
	nodes := []string{
		"load-balancer",
		"kubernetes-node",
	}
	return nodes, nil
}

func (m *MockHypervisor) ListAvailableImages(vms hope.VMs) ([]string, error) {
	nodes := []string{
		"load-balancer",
		"kubernetes-node",
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

func (m *MockHypervisor) CopyImageMode() hypervisors.CopyImageMode {
	return hypervisors.CopyImageModeNone
}

func (m *MockHypervisor) CopyImage(a hope.VMs, b hope.VMImageSpec, c hypervisors.Hypervisor) error {
	return hypervisors.ErrCopyImageNotImplemented
}

func (m *MockHypervisor) CreateImage(a hope.VMs, b hope.VMImageSpec, c []string, d bool) error {
	return nil
}

func (m *MockHypervisor) CreateNode(a hope.Node, b hope.VMs, c hope.VMImageSpec) error {
	return nil
}

func (m *MockHypervisor) StartVM(string) error {
	return nil
}

func (m *MockHypervisor) StopVM(string) error {
	return nil
}

func (m *MockHypervisor) DeleteVM(string) error {
	return nil
}

func (m *MockHypervisor) VMIPAddress(string) (string, error) {
	return "192.168.1.5", nil
}

func toHypervisorStub(node hope.Node) (hypervisors.Hypervisor, error) {
	if !node.IsHypervisor() {
		return nil, fmt.Errorf("Not a hypervisor")
	}
	return &MockHypervisor{node}, nil
}

// Implemented as a suite to allow manipulating the hypervisor factory
// function.
type NodesTestSuite struct {
	suite.Suite

	originalToHypervisor hypervisors.ToHypervisorFactoryFunc
}

func (s *NodesTestSuite) SetupTest() {
	s.originalToHypervisor = hypervisors.ToHypervisor
	hypervisors.ToHypervisor = toHypervisorStub
}

func (s *NodesTestSuite) TeardownTest() {
	hypervisors.ToHypervisor = s.originalToHypervisor
}

// Actual test method to run the suite
func TestNodes(t *testing.T) {
	suite.Run(t, new(NodesTestSuite))
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
// this extensively quite yet.
func (s *NodesTestSuite) TestGetNodes() {
	t := s.T()
	resetViper(t)

	nodes, err := getNodes()
	assert.Nil(t, err)
	assert.Equal(t, testNodes, nodes)
}

func (s *NodesTestSuite) TestGetNodeNames() {
	t := s.T()
	resetViper(t)

	var tests = []struct {
		name      string
		roles     []string
		nodeNames []string
	}{
		{"Hypervisors", []string{hope.NodeRoleHypervisor.String()}, []string{"beast1"}},
		{"Load Balancers", []string{hope.NodeRoleLoadBalancer.String()}, []string{"test-load-balancer"}},
		{"Masters", []string{hope.NodeRoleMaster.String()}, []string{"test-master-01", "test-master-02", "test-master-03"}},
		{"Nodes", []string{hope.NodeRoleNode.String()}, []string{"test-node-01"}},
		{"Masters and Nodes", []string{hope.NodeRoleMaster.String(), hope.NodeRoleNode.String()}, []string{"test-master-01", "test-master-02", "test-master-03", "test-node-01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names, err := GetNodeNames(tt.roles)
			assert.Nil(t, err)
			assert.Equal(t, tt.nodeNames, names)
		})
	}
}

func (s *NodesTestSuite) TestGetNode() {
	t := s.T()
	resetViper(t)

	expected := worker1Node
	expected.Host = "test-node-01"
	expected.Hypervisor = ""

	node, err := GetNode("test-node-01")
	assert.Nil(t, err)

	assert.Equal(t, expected, node)

	_, err = GetNode("sets-node-01")
	assert.Equal(t, "failed to find node: sets-node-01", err.Error())
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

	expected := master1Node
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
	assert.NoError(t, err)
	assert.Equal(t, beast1Node, node)
}

func (s *NodesTestSuite) TestGetHypervisor() {
	t := s.T()
	resetViper(t)

	expected := beast1Node

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
	assert.Equal(t, "failed to find a hypervisor named sets-node-01", err.Error())
}

func (s *NodesTestSuite) TestGetAvailableMasters() {
	t := s.T()
	resetViper(t)

	expectedOrig := []hope.Node{master1Node, master2Node, master3Node}
	expected := []hope.Node{}

	for i, n := range expectedOrig {
		if i == 1 {
			n.Host = "192.168.1.10"
		} else {
			n.Host = fmt.Sprintf("test-master-0%d", i+1)
		}

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

	expected := loadBalancerNode
	expected.Host = "test-load-balancer"
	expected.Hypervisor = ""

	node, err := GetLoadBalancer()
	assert.Nil(t, err)

	assert.Equal(t, node, expected)
}
