package utils

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/esxi"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

func getNodes() (*[]hope.Node, error) {
	var nodes []hope.Node
	err := viper.UnmarshalKey("nodes", &nodes)

	nameMap := map[string]bool{}
	for _, node := range nodes {
		if _, ok := nameMap[node.Name]; ok {
			return nil, fmt.Errorf("Multiple nodes found in configuration file named: %s", node.Name)
		}
		nameMap[node.Name] = true
	}

	return &nodes, err
}

func GetNode(name string) (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.Name == name {
			return expandHypervisor(&node)
		}
	}

	return nil, fmt.Errorf("Failed to find a node named %s", name)
}

// HasNode -- Check whether a node has been defined in the hope file, even if
//   the node doesn't exist on its hypervisor yet.
func HasNode(name string) bool {
	nodes, err := getNodes()
	if err != nil {
		return false
	}

	for _, node := range *nodes {
		if node.Name == name {
			return true
		}
	}

	return false
}

func GetAnyMaster() (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsMaster() {
			return expandHypervisor(&node)
		}
	}

	return nil, errors.New("Failed to find any master in nodes config")
}

func GetHypervisors() (*[]hope.Node, error) {
	retVal := []hope.Node{}
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsHypervisor() {
			retVal = append(retVal, node)
		}
	}

	return &retVal, nil
}

func expandHypervisor(node *hope.Node) (*hope.Node, error) {
	if node.Hypervisor == "" {
		return node, nil
	}

	hypervisor, err := GetHypervisor(node.Hypervisor)
	if err != nil {
		return nil, err
	}

	ip, err := esxi.GetIpAddressOfVmNamed(hypervisor.ConnectionString(), node.Name)
	if err != nil {
		return nil, err
	}

	ip = strings.TrimSpace(ip)
	if ip == "0.0.0.0" {
		return nil, fmt.Errorf("Failed to find IP for vm %s on %s", node.Name, hypervisor.Name)
	}

	newNode := *node
	newNode.Hypervisor = ""
	newNode.Host = ip
	return &newNode, nil
}

func GetHypervisor(name string) (*hope.Node, error) {
	// Any nice way to generalize this?
	// Copied from GetNode
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.Name == name {
			if node.IsHypervisor() {
				return &node, nil
			}

			return nil, fmt.Errorf("Node named %s is not a hypervisor", name)
		}
	}

	return nil, fmt.Errorf("Failed to find a hypervisor named %s", name)
}

func GetMasters() (*[]hope.Node, error) {
	retVal := []hope.Node{}
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsMaster() {
			exNode, err := expandHypervisor(&node)
			if err != nil {
				return nil, err
			}
			retVal = append(retVal, *exNode)
		}
	}

	return &retVal, nil
}

func KubectlFromAnyMaster() (*kubeutil.Kubectl, error) {
	// To prevent "dereferencing" all the master nodes in advance, and making
	//   a ton of extra network traffic, do them incrementally until a valid
	//   kubeconfig is found.
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if !node.IsMaster() {
			continue
		}

		nNode, err := expandHypervisor(&node)
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

func GetLoadBalancer() (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsLoadBalancer() {
			return &node, nil
		}
	}

	// This feels dirty, and a little broken.
	// Maybe need a dedicated NodeNotFound kind of error that can be handled
	//   independently of other errors if desired.
	return nil, nil
}
