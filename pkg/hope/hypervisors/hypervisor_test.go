package hypervisors

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGetBuildPlan(t *testing.T) {
	hv1, err := ToHypervisor(exampleEsxiHypervisorNode1)
	assert.NoError(t, err)
	hv2, err := ToHypervisor(exampleEsxiHypervisorNode2)
	assert.NoError(t, err)

	plans, err := GetEnginePlans([]Hypervisor{hv1, hv2})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(plans))

	plan := plans[0]
	assert.Equal(t, "esxi", plan.Engine)
	assert.Equal(t, 2, plan.NumHypervisors)
	assert.Equal(t, []Hypervisor{hv1}, plan.BuildHypervisors)
	assert.Equal(t, []Hypervisor{hv1, hv2}, plan.CopyHypervisors)
}
