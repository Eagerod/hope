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

func TestHelpExecutes(t *testing.T) {
	var tests = []struct {
		name string
		args []string
	}{
		{"Base Command", []string{}},
		{"Node Base Command", []string{"node"}},
		{"Node Hostname", []string{"node", "hostname"}},
		{"Node Init", []string{"node", "init"}},
		{"Node Reset", []string{"node", "reset"}},
		{"Node SSH", []string{"node", "ssh"}},
		{"Unifi Base Command", []string{"unifi"}},
		{"Unifi Access Point", []string{"unifi", "ap"}},
		{"Deploy", []string{"deploy"}},
		{"Kubeconfig", []string{"kubeconfig"}},
		{"List", []string{"list"}},
		{"Remove", []string{"remove"}},
		{"Run", []string{"run"}},
		{"Shell", []string{"shell"}},
		{"Token", []string{"token"}},
		{"Version", []string{"version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allArgs := append(tt.args, "--help")
			cmd := exec.Command(bin, allArgs...)
			_, err := cmd.CombinedOutput()
			assert.NoError(t, err)
		})
	}
}
