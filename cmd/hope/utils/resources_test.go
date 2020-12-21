package utils

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func TestGetResources(t *testing.T) {
	resetViper(t)

	resources, err := GetResources()
	assert.Nil(t, err)
	assert.Equal(t, 6, len(*resources))
}

func TestGetIdentifiableResources(t *testing.T) {
	resetViper(t)

	var tests = []struct {
		name     string
		names    []string
		tags     []string
		expected int
	}{
		{"No matches", []string{}, []string{}, 0},
		{"Only name", []string{"calico"}, []string{}, 1},
		{"Multiple names", []string{"calico", "load-balancer-config"}, []string{}, 2},
		{"Only tag", []string{}, []string{"network"}, 2},
		{"Multiple tags", []string{}, []string{"network", "database"}, 4},
		{"Tag and name", []string{"calico"}, []string{"database"}, 3},
		{"Tag and name overlap", []string{"calico"}, []string{"network"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := GetIdentifiableResources(&tt.names, &tt.tags)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, len(*resources))
		})
	}
}
