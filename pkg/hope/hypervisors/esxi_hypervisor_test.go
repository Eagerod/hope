package hypervisors

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
)

// Implemented as a suite to allow manipulating the ExecSCP + ExecSSH
// functions.
type EsxiHypervisorTestSuite struct {
	suite.Suite

	oldExecSSH ssh.ExecSSHFunc
	oldExecSCP scp.ExecSCPFunc

	vms            hope.VMs
	hypervisorNode hope.Node
}

func (s *EsxiHypervisorTestSuite) SetupTest() {
	s.oldExecSSH = ssh.ExecSSH
	s.oldExecSCP = scp.ExecSCP

	s.vms = hope.VMs{
		Cache:  "/var/lib/packer/cache",
		Output: "/var/lib/packer/images",
		Root:   "../../../vms",
		Images: []hope.VMImageSpec{
			hope.VMImageSpec{
				Name:        "some-image",
				Hypervisors: []string{"beast1"},
				Parameters:  []string{},
			},
		},
	}

	s.hypervisorNode = hope.Node{
		Name:      "beast1",
		Role:      "hypervisor",
		Engine:    "esxi",
		Host:      "192.168.10.40",
		User:      "root",
		Datastore: "Main",
		Network:   "VM Network",
	}
}

func (s *EsxiHypervisorTestSuite) TeardownTest() {
	ssh.ExecSSH = s.oldExecSSH
	scp.ExecSCP = s.oldExecSCP
}

// Actual test method to run the suite
func TestEsxiHypervisor(t *testing.T) {
	suite.Run(t, new(EsxiHypervisorTestSuite))
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
// this extensively quite yet.
func (s *EsxiHypervisorTestSuite) TestCopyImage() {
	t := s.T()

	sshExecutions := 0
	scpExecutions := 0

	scp.ExecSCP = func(args ...string) error {
		scpExecutions += 1
		assert.Equal(t, args, []string{
			"-pr",
			"/var/lib/packer/images/some-image",
			"root@192.168.10.40:/vmfs/volumes/Main/ovfs/some-image",
		})
		return nil
	}

	ssh.ExecSSH = func(args ...string) error {
		sshExecutions += 1
		assert.Equal(t, args, []string{
			"root@192.168.10.40",
			"rm",
			"-rf",
			"/vmfs/volumes/Main/ovfs/some-image",
		})
		return nil
	}

	esxi, err := ToHypervisor(s.hypervisorNode)
	assert.NoError(t, err)

	err = esxi.CopyImage(s.vms, s.vms.Images[0], esxi)
	assert.NoError(t, err)

	assert.Equal(t, 1, scpExecutions)
	assert.Equal(t, 1, sshExecutions)
}
