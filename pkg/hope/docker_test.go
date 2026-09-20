package hope

import (
	"errors"
	"testing"
)

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

import (
	"github.com/Eagerod/hope/pkg/docker"
)

func GetDockerNotExpected() docker.GetDockerFunc {
	return func(args ...string) (string, error) {
		return "", errors.New("GetDocker unexpectedly called")
	}
}

func ExecDockerNotExpected() docker.ExecDockerFunc {
	return func(args ...string) error {
		return errors.New("ExecDocker unexpectedly called")
	}
}

type DoDockerBuildTestSuite struct {
	suite.Suite

	originalGetDocker  func(args ...string) (string, error)
	originalExecDocker func(args ...string) error
}

func (s *DoDockerBuildTestSuite) SetupTest() {
	s.originalGetDocker = docker.GetDocker
	s.originalExecDocker = docker.ExecDocker
}

func (s *DoDockerBuildTestSuite) TeardownTest() {
	docker.GetDocker = s.originalGetDocker
	docker.ExecDocker = s.originalExecDocker
}

// Actual test method to run the suite.
func TestDoDockerBuild(t *testing.T) {
	suite.Run(t, new(DoDockerBuildTestSuite))
}

func (s *DoDockerBuildTestSuite) TestPathAndSourceAreMutuallyExclusive() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path:   "./image",
			Source: "ubuntu:24.04",
			Tag:    "registry.example.com/test:latest",
		},
	}

	docker.GetDocker = GetDockerNotExpected()
	docker.ExecDocker = ExecDockerNotExpected()

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.Error(t, err)
}

func (s *DoDockerBuildTestSuite) TestUnknownPullConstraint() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "sometimes",
		},
	}

	docker.GetDocker = GetDockerNotExpected()
	docker.ExecDocker = ExecDockerNotExpected()

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.Error(t, err)
}

func (s *DoDockerBuildTestSuite) TestBuildIfNotPresentImageExists() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/test:latest", "--format={{.Repository}}:{{.Tag}}",
		})
		return "example/test:latest\nother/image:latest\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"push", "example/test:latest"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestBuildIfNotPresentImageMissingPullSucceeds() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/test:latest", "--format={{.Repository}}:{{.Tag}}",
		})
		return "other/image:latest\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/test:latest"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestBuildIfNotPresentImageMissingPullFails() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/test:latest", "--format={{.Repository}}:{{.Tag}}",
		})
		return "other/image:latest\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/test:latest"})
			execDockerCall += 1
			return errors.New("Error response from daemon: manifest for example/test:latest not found: manifest unknown: manifest unknown")
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/test:latest"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestBuildIfNotPresentImageMissingUnspecifiedTagUsesLatest() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test",
			Pull: "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/test", "--format={{.Repository}}:{{.Tag}}",
		})
		return "other/image:latest\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/test"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/test"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestBuildAlwaysPulls() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "always",
		},
	}

	docker.GetDocker = GetDockerNotExpected()

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/test:latest"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestBuildIfNotPresentWithEmptyDockerOutput() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Path: "./image",
			Tag:  "example/test:latest",
			Pull: "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/test:latest", "--format={{.Repository}}:{{.Tag}}",
		})
		return "", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"build", "./image", "-t", "example/test:latest"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/test:latest"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestCacheSourceFound() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Source: "example/source:1.2.3",
			Tag:    "example/target:1.2.3",
			Pull:   "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/source:1.2.3", "--format={{.Repository}}:{{.Tag}}",
		})
		return "example/source:1.2.3\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"tag", "example/source:1.2.3", "example/target:1.2.3"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"push", "example/target:1.2.3"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}

func (s *DoDockerBuildTestSuite) TestCacheSourceMissing() {
	t := s.T()

	resource := Resource{
		Name: "test-resource",
		Build: BuildSpec{
			Source: "example/source:1.2.3",
			Tag:    "example/target:1.2.3",
			Pull:   "if-not-present",
		},
	}

	docker.GetDocker = func(args ...string) (string, error) {
		assert.Equal(t, args, []string{
			"images", "example/source:1.2.3", "--format={{.Repository}}:{{.Tag}}",
		})
		return "other/image:latest\n", nil
	}

	execDockerCall := 1
	docker.ExecDocker = func(args ...string) error {
		if execDockerCall == 1 {
			assert.Equal(t, args, []string{"pull", "example/source:1.2.3"})
			execDockerCall += 1
		} else if execDockerCall == 2 {
			assert.Equal(t, args, []string{"tag", "example/source:1.2.3", "example/target:1.2.3"})
			execDockerCall += 1
		} else if execDockerCall == 3 {
			assert.Equal(t, args, []string{"push", "example/target:1.2.3"})
			execDockerCall += 1
		} else {
			return errors.New("Unexpected ExecDocker call")
		}
		return nil
	}

	err := DoDockerBuild(logrus.New().WithFields(logrus.Fields{}), resource)
	assert.NoError(s.T(), err)
}
