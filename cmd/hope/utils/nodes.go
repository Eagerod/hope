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
	"github.com/Eagerod/hope/pkg/sliceutil"
)

var toHypervisor func(hope.Node) (hypervisors.Hypervisor, error) = hypervisors.ToHypervisor

func getNodes() ([]hope.Node, error) {
	var nodes []hope.Node
	err := viper.UnmarshalKey("nodes", &nodes)

	nameMap := map[string]bool{}
	for _, node := range nodes {
		if _, ok := nameMap[node.Name]; ok {
			return nil, fmt.Errorf("Multiple nodes found in configuration file named: %s", node.Name)
		}
		nameMap[node.Name] = true
	}

	return nodes, err
}

func GetNodeNames(types []string) ([]string, error) {
	typesMap := map[string]bool{}
	for _, t := range types {
		typesMap[t] = true
	}

	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	nodeNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := typesMap[node.Role]; ok {
			nodeNames = append(nodeNames, node.Name)
		}
	}

	return nodeNames, nil
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

	return hope.Node{}, fmt.Errorf("Failed to find a node named %s", name)
}

// HasNode -- Check whether a node has been defined in the hope file, even if
//   the node doesn't exist on its hypervisor yet.
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

	return hope.Node{}, errors.New("Failed to find any master in nodes config")
}

func GetHypervisors() ([]hypervisors.Hypervisor, error) {
	retVal := []hypervisors.Hypervisor{}

	nodes, err := getNodes()
	if err != nil {
		return retVal, err
	}

	for _, node := range nodes {
		if node.IsHypervisor() {
			hypervisor, err := toHypervisor(node)
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
			return toHypervisor(node)
		}
	}

	return nil, fmt.Errorf("Failed to find a hypervisor named %s", name)
}

// GetAvailableMasters -- Returns the list of master nodes that can be reached
//   in one way or another.
// Doesn't confirm if the masters are configured, or are in the load balanced
//   set of masters; only that the node exists on its defined hypervisor.
func GetAvailableMasters() ([]hope.Node, error) {
	retVal := []hope.Node{}
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.IsMaster() {
			hv, err := GetHypervisor(node.Hypervisor)
			if err != nil {
				return nil, err
			}

			hvNodes, err := hv.ListNodes()
			if err != nil {
				return nil, err
			}

			if sliceutil.StringInSlice(node.Name, hvNodes) {
				exNode, err := expandHypervisor(node)
				if err != nil {
					return nil, err
				}
				retVal = append(retVal, exNode)
			}
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

	return nil, errors.New("Failed to find a kubeconfig file in any of the master nodes")
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

	// This feels dirty, and a little broken.
	// Maybe need a dedicated NodeNotFound kind of error that can be handled
	//   independently of other errors if desired.
	return hope.Node{}, nil
}

func HypervisorForNodeNamed(name string) (*hypervisors.Hypervisor, error) {
	node, err := GetBareNode(name)
	if err != nil {
		return nil, err
	}

	hypervisor, err := GetHypervisor(node.Hypervisor)
	return &hypervisor, err
}
