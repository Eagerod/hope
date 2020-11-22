package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

var bin string = filepath.Join(".", "build", "hope")
var commands []string = []string{
	"deploy",
	"hostname",
	"init",
	"reset",
	"version",
}

func TestHelpExecutes(t *testing.T) {
	cmd := exec.Command(bin, "--help")

	_, err := cmd.CombinedOutput()
	assert.NoError(t, err)

	for _, command := range commands {
		cmd = exec.Command(bin, command, "--help")

		_, err = cmd.CombinedOutput()
		assert.NoError(t, err)
	}
}
