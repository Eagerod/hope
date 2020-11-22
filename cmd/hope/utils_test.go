package cmd

import (
	"os"
	"testing"
)

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestReplaceParametersInString(t *testing.T) {
	os.Setenv("HELLO", "Hello,")
	os.Setenv("WORLD", "World!")

	var tests = []struct {
		name       string
		in         string
		out        string
		parameters []string
	}{
		{"All Envs", "${HELLO} $WORLD", "Hello, World!", []string{"HELLO", "WORLD"}},
		{"One Env", "${HELLO} $WORLD", "Hello, Moon!", []string{"HELLO", "WORLD=Moon!"}},
		{"No Envs", "${HELLO} $WORLD", "${HELLO} $WORLD", []string{}},
		{"Var with =", "${HELLO} $WORLD", "e30= $WORLD", []string{"HELLO=e30="}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := replaceParametersInString(tt.in, tt.parameters)
			assert.Nil(t, err)
			assert.Equal(t, tt.out, s)
		})
	}

	os.Unsetenv("HELLO")
	os.Unsetenv("WORLD")
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func TestGetResources(t *testing.T) {
	// Assume config file in the project root.
	// Probably bad practice, but better test than having nothing at all.
	viper.AddConfigPath("../../")
	viper.SetConfigName("hope")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	assert.Nil(t, err)

	resources, err := getResources()
	assert.Nil(t, err)
	assert.Equal(t, 5, len(*resources))
}

func TestGetIdentifiableResources(t *testing.T) {
	// Assume config file in the project root.
	// Probably bad practice, but better test than having nothing at all.
	viper.AddConfigPath("../../")
	viper.SetConfigName("hope")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	assert.Nil(t, err)

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
			resources, err := getIdentifiableResources(&tt.names, &tt.tags)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, len(*resources))
		})
	}
}
