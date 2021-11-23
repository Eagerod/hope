package hope

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestResourceType(t *testing.T) {
	var tests = []struct {
		name   string
		value  ResourceType
		strval string
	}{
		{"ResourceTypeFile", ResourceTypeFile, "file"},
		{"ResourceTypeInline", ResourceTypeInline, "inline"},
		{"ResourceTypeDockerBuild", ResourceTypeDockerBuild, "docker"},
		{"ResourceTypeJob", ResourceTypeJob, "job"},
		{"ResourceTypeExec", ResourceTypeExec, "exec"},
		{"Improper ResourceType", 25, "%!ResourceType(25)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.strval, tt.value.String())
		})
	}
}

func TestNodeRole(t *testing.T) {
	var tests = []struct {
		name   string
		value  NodeRole
		strval string
	}{
		{"NodeRoleHypervisor", NodeRoleHypervisor, "hypervisor"},
		{"NodeRoleLoadBalancer", NodeRoleLoadBalancer, "load-balancer"},
		{"NodeRoleMaster", NodeRoleMaster, "master"},
		{"NodeRoleMasterAndNode", NodeRoleMasterAndNode, "master+node"},
		{"NodeRoleNode", NodeRoleNode, "node"},
		{"Improper NodeRole", 25, "%!NodeRole(25)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.strval, tt.value.String())
		})
	}
}
