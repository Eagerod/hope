package utils

import (
	"errors"
	"fmt"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/hope/hypervisors"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

type NodeNotFoundError struct {
	node string
}

func NewNodeNotFoundError(node string) error {
	return NodeNotFoundError{node}
}

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("failed to find node: %s", e.node)
}

func getNodes() ([]hope.Node, error) {
	var nodes []hope.Node
	err := viper.UnmarshalKey("nodes", &nodes)

	nameMap := map[string]bool{}
	for _, node := range nodes {
		if _, ok := nameMap[node.Name]; ok {
			return nil, fmt.Errorf("multiple nodes found in configuration file named: %s", node.Name)
		}
		nameMap[node.Name] = true
	}

	return nodes, err
}

func GetNodeNames(types []string) ([]string, error) {
	nodes, err := GetBareNodeTypes(types)
	if err != nil {
		return nil, err
	}

	rv := make([]string, 0, len(nodes))
	for _, node := range nodes {
		rv = append(rv, node.Name)
	}

	return rv, nil
}

func GetNode(name string) (hope.Node, error) {
	node, err := GetBareNode(name)
	if err != nil {
		return hope.Node{}, err
	}

	return expandHypervisor(node)
}

func GetBareNode(name string) (hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return hope.Node{}, err
	}

	for _, node := range nodes {
		if node.Name == name {
			return node, nil
		}
	}

	return hope.Node{}, NewNodeNotFoundError(name)
}

func GetBareNodeTypes(types []string) ([]hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	typesMap := map[string]bool{}
	for _, t := range types {
		typesMap[t] = true
	}

	rv := make([]hope.Node, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := typesMap[node.Role]; ok {
			rv = append(rv, node)
		}
	}

	return rv, nil
}

// HasNode -- Check whether a node has been defined in the hope file, even if
// the node doesn't exist on its hypervisor yet.
func HasNode(name string) bool {
	nodes, err := getNodes()
	if err != nil {
		return false
	}

	for _, node := range nodes {
		if node.Name == name {
			return true
		}
	}

	return false
}

func GetAnyMaster() (hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return hope.Node{}, err
	}

	for _, node := range nodes {
		if node.IsMaster() {
			return expandHypervisor(node)
		}
	}

	return hope.Node{}, errors.New("failed to find any master in nodes config")
}

func GetHypervisors() ([]hypervisors.Hypervisor, error) {
	retVal := []hypervisors.Hypervisor{}

	nodes, err := getNodes()
	if err != nil {
		return retVal, err
	}

	for _, node := range nodes {
		if node.IsHypervisor() {
			hypervisor, err := hypervisors.ToHypervisor(node)
			if err != nil {
				return nil, err
			}
			retVal = append(retVal, hypervisor)
		}
	}

	return retVal, nil
}

func expandHypervisor(node hope.Node) (hope.Node, error) {
	if node.Hypervisor == "" {
		return node, nil
	}

	hypervisor, err := GetHypervisor(node.Hypervisor)
	if err != nil {
		return hope.Node{}, err
	}

	return hypervisor.ResolveNode(node)
}

func GetHypervisor(name string) (hypervisors.Hypervisor, error) {
	// Any nice way to generalize this?
	// Copied from GetNode
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.Name == name {
			return hypervisors.ToHypervisor(node)
		}
	}

	return nil, fmt.Errorf("failed to find a hypervisor named %s", name)
}

// GetAvailableMasters -- Returns the list of master nodes that can be reached
// in one way or another.
// Doesn't confirm if the masters are configured, or are in the load balanced
// set of masters; only that the node exists on its defined hypervisor.
func GetAvailableMasters() ([]hope.Node, error) {
	retVal := []hope.Node{}
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if !node.IsMaster() {
			continue
		}

		if node.Hypervisor == "" {
			retVal = append(retVal, node)
			continue
		}

		// Need to get more node details from hypervisor
		hv, err := GetHypervisor(node.Hypervisor)
		if err != nil {
			return nil, err
		}

		hvHasNode, err := hypervisors.HasNode(hv, node.Name)
		if err != nil {
			return nil, err
		}

		if hvHasNode {
			exNode, err := expandHypervisor(node)
			if err != nil {
				return nil, err
			}
			retVal = append(retVal, exNode)
		}
	}

	return retVal, nil
}

func KubectlFromAnyMaster() (*kubeutil.Kubectl, error) {
	// To prevent "dereferencing" all the master nodes in advance, and making
	//   a ton of extra network traffic, do them incrementally until a valid
	//   kubeconfig is found.
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if !node.IsMaster() {
			continue
		}

		nNode, err := expandHypervisor(node)
		if err != nil {
			return nil, err
		}

		kubectl, err := kubeutil.NewKubectlFromNode(nNode.ConnectionString())
		if err == nil {
			return kubectl, nil
		}
	}

	return nil, errors.New("failed to find a kubeconfig file in any of the master nodes")
}

func GetLoadBalancer() (hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return hope.Node{}, err
	}

	for _, node := range nodes {
		if node.IsLoadBalancer() {
			return expandHypervisor(node)
		}
	}

	return hope.Node{}, NewNodeNotFoundError("load-balancer")
}

func HypervisorForNodeNamed(name string) (*hypervisors.Hypervisor, error) {
	node, err := GetBareNode(name)
	if err != nil {
		return nil, err
	}

	hypervisor, err := GetHypervisor(node.Hypervisor)
	return &hypervisor, err
}
