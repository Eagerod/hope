package hope

import (
	"testing"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

// Implemented as a suite to allow manipulating kubeutil.kubectl funcs
type KubectlJobsTestSuite struct {
	suite.Suite

	originalGetKubectl func(kubectl *kubeutil.Kubectl, args ...string) (string, error) 
}

func (s *KubectlJobsTestSuite) SetupTest() {
	s.originalGetKubectl = kubeutil.GetKubectl

}

func (s *KubectlJobsTestSuite) TeardownTest() {
	kubeutil.GetKubectl = s.originalGetKubectl
}

// Actual test method to run the suite
func TestKubectlJobs(t *testing.T) {
	suite.Run(t, new(KubectlJobsTestSuite))
}

func (s *KubectlJobsTestSuite) TestGetJobStatus() {
	t := s.T()

	kubeutil.GetKubectl = func(kubectl *kubeutil.Kubectl, args ...string) (string, error) {
		return "SuccessCriteriaMet\nComplete", nil
	}

	kubectl := kubeutil.Kubectl{}
	status, err := GetJobStatus(log.WithFields(log.Fields{}), &kubectl, "default", "imaginary-job")
	assert.Nil(t, err)
	assert.Equal(t, status, JobStatusComplete)
}
