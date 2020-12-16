package node

import (
	"testing"
)

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
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

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func TestGetNodes(t *testing.T) {
	resetViper(t)

	nodes, err := getNodes()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(*nodes))

	expected := []hope.Node{
		hope.Node{
			Name: "home-master-01",
			Role: "master",
			Host: "192.168.1.31",
			User: "root",
		},
		hope.Node{
			Name: "home-node-01",
			Role: "node",
			Host: "192.168.1.30",
			User: "root",
		},
	}

	assert.Equal(t, *nodes, expected)
}

func TestGetNode(t *testing.T) {
	resetViper(t)

	nodes, err := getNodes()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(*nodes))

	var tests = []struct {
		name     string
		nodeName string
		node     hope.Node
	}{
		{"Get home-master-01", "home-master-01", hope.Node{Name: "home-master-01", Role: "master", Host: "192.168.1.31", User: "root"}},
		{"Get home-node-01", "home-node-01", hope.Node{Name: "home-node-01", Role: "node", Host: "192.168.1.30", User: "root"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := getNode(tt.nodeName)
			assert.Nil(t, err)
			assert.Equal(t, tt.node, *node)
		})
	}
}
