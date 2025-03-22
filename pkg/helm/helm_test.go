package helm

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Implemented as a suite to allow manipulating the helm wrapper func.
type HelmTestSuite struct {
	suite.Suite

	originalExecHelm ExecHelmFunc
	originalGetHelm  GetHelmFunc
}

func (s *HelmTestSuite) SetupTest() {
	s.originalExecHelm = ExecHelm
	s.originalGetHelm = GetHelm
}

func (s *HelmTestSuite) TeardownTest() {
	ExecHelm = s.originalExecHelm
	GetHelm = s.originalGetHelm
}

// Actual test method to run the suite
func TestHelm(t *testing.T) {
	suite.Run(t, new(HelmTestSuite))
}

func (s *HelmTestSuite) TestHasRepo() {
	t := s.T()

	r := ""
	GetHelm = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{"repo", "list"})
		return r, nil
	}

	r = "NAME     	URL\ntest	https://example.com/charts/"
	hasRepo, err := HasRepo("test", "https://example.com/charts/")
	assert.NoError(t, err)
	assert.True(t, hasRepo)

	r = "NAME     	URL\ntest	https://example.com/charts/"
	hasRepo, err = HasRepo("test", "https://example.com/charts")
	assert.NoError(t, err)
	assert.True(t, hasRepo)

	r = "NAME     	URL\ntest	https://example.com/charts"
	hasRepo, err = HasRepo("test", "https://example.com/charts")
	assert.NoError(t, err)
	assert.True(t, hasRepo)

	r = "NAME     	URL\ntest	https://example.com/charts"
	hasRepo, err = HasRepo("test", "https://example.com/charts/")
	assert.NoError(t, err)
	assert.True(t, hasRepo)

	r = "NAME     	URL"
	hasRepo, err = HasRepo("test", "https://example.com/charts/")
	assert.NoError(t, err)
	assert.False(t, hasRepo)

	r = "NAME     	URL\ntest	https://example.com/strahc"
	hasRepo, err = HasRepo("test", "https://example.com/charts")
	if assert.Error(t, err) {
		assert.Equal(t, "local helm repo 'test' has a different url than expected", err.Error())
	}
	assert.False(t, hasRepo)
}
