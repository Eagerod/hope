package utils

import (
	"testing"

	"github.com/Eagerod/hope/pkg/hope"
	"github.com/stretchr/testify/assert"
)

var testJobs []hope.Job = []hope.Job{
	{
		Name:       "test-job",
		File:       "test/job.yaml",
		Parameters: []string{"LOG_LINE=I did the thing"},
	},
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
// this extensively quite yet.
func TestGetJobs(t *testing.T) {
	resetViper(t)

	jobs, err := GetJobs()
	assert.Nil(t, err)
	assert.Equal(t, testJobs, *jobs)
}

func TestGetJob(t *testing.T) {
	resetViper(t)

	job, err := GetJob("test-job")
	assert.Nil(t, err)
	assert.Equal(t, testJobs[0], *job)
}
