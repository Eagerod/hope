package hope

import (
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestSubstituteTextFromEnv(t *testing.T) {
	os.Setenv("HELLO", "Hello,")
	os.Setenv("WORLD", "World!")

	var tests = []struct {
		name string
		in  string
		out  string
		vars  []string
	}{
		{"All Envs", "${HELLO} $WORLD", "Hello, World!", []string{"HELLO", "WORLD"}},
		{"One Env", "${HELLO} $WORLD", "Hello, $WORLD", []string{"HELLO"}},
		{"No Envs", "${HELLO} $WORLD", "${HELLO} $WORLD", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTextSubstitutorFromString(tt.in)
			err := ts.SubstituteTextFromEnv(tt.vars)
			assert.Nil(t, err)
			assert.Equal(t, tt.out, string(*ts.Bytes))
		})
	}

	os.Unsetenv("HELLO")
	os.Unsetenv("WORLD")
}

func TestSubstituteTextFromVars(t *testing.T) {
	// Set the envs to make sure they aren't being pulled from there.
	os.Setenv("HELLO", "Goodnight,")
	os.Setenv("WORLD", "Moon!")

	var tests = []struct {
		name string
		in  string
		out  string
		vars  map[string]string
	}{
		{"All Envs", "${HELLO} $WORLD", "Hello, World!", map[string]string{"HELLO": "Hello,", "WORLD": "World!"}},
		{"One Env", "${HELLO} $WORLD", "Hello, $WORLD", map[string]string{"HELLO": "Hello,"}},
		{"No Envs", "${HELLO} $WORLD", "${HELLO} $WORLD", map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTextSubstitutorFromString(tt.in)
			err := ts.SubstituteTextFromMap(tt.vars)
			assert.Nil(t, err)
			assert.Equal(t, tt.out, string(*ts.Bytes))
		})
	}

	os.Unsetenv("HELLO")
	os.Unsetenv("WORLD")
}
