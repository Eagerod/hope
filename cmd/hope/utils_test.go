package cmd

import (
	"os"
	"testing"
)

import (
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
