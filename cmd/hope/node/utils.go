package node

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

// Copied directly from the parent package.
// Need to find the canonical way of distributing cmd-only code to nested
//   packages
func getNodes() (*[]hope.Node, error) {
	var nodes []hope.Node
	err := viper.UnmarshalKey("nodes", &nodes)

	nameMap := map[string]bool{}
	for _, node := range nodes {
		if _, ok := nameMap[node.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Multiple nodes found in configuration file named: %s", node.Name))
		}
		nameMap[node.Name] = true
	}

	return &nodes, err
}

func getNode(name string) (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.Name == name {
			return &node, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("Failed to find a node named %s", name))
}
