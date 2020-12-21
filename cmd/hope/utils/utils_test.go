package utils

import (
	"os"
	"testing"
)

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
			s, err := ReplaceParametersInString(tt.in, tt.parameters)
			assert.Nil(t, err)
			assert.Equal(t, tt.out, s)
		})
	}

	os.Unsetenv("HELLO")
	os.Unsetenv("WORLD")
}
