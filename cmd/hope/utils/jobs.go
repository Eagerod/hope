package utils

import (
	"fmt"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

func GetJobs() (*[]hope.Job, error) {
	var jobs []hope.Job
	err := viper.UnmarshalKey("jobs", &jobs)

	nameMap := map[string]bool{}
	for _, job := range jobs {
		if _, ok := nameMap[job.Name]; ok {
			return nil, fmt.Errorf("Multiple jobs found in configuration file named: %s", job.Name)
		}
		nameMap[job.Name] = true
	}

	return &jobs, err
}

func GetJob(jobName string) (*hope.Job, error) {
	jobs, err := GetJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range *jobs {
		if job.Name == jobName {
			return &job, nil
		}
	}

	return nil, fmt.Errorf("Failed to find a job named %s", jobName)
}
