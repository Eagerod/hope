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
)

func GetNodes() (*[]hope.Node, error) {
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
	nodes, err := GetNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.Name == name {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("Failed to find a node named %s", name)
}

func GetAnyMaster() (*hope.Node, error) {
	nodes, err := GetNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsMaster() {
			return &node, nil
		}
	}

	return nil, errors.New("Failed to find any master in nodes config")
}

func GetHypervisors() (*[]hope.Node, error) {
	retVal := []hope.Node{}
	nodes, err := GetNodes()
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

func GetHypervisor(name string) (*hope.Node, error) {
	// Any nice way to generalize this?
	// Copied from GetNode
	nodes, err := GetNodes()
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
