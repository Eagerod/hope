package cmd

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

const (
	ResourceTypeFile        string = "file"
	ResourceTypeInline      string = "inline"
	ResourceTypeDockerBuild string = "build"
	ResourceTypeJob         string = "job"
)

// Should be defined in hope pkg
type BuildSpec struct {
	Path string
	Tag  string
}

type Resource struct {
	Name   string
	File   string
	Inline string
	Build  BuildSpec
	Job    string
}

type Job struct {
	Name       string
	File       string
	Parameters []string
}

func (resource *Resource) GetType() (string, error) {
	detectedTypes := []string{}
	if len(resource.File) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeFile)
	}
	if len(resource.Inline) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeInline)
	}
	if len(resource.Build.Path) != 0 && len(resource.Build.Tag) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeDockerBuild)
	}
	if len(resource.Job) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeJob)
	}

	switch len(detectedTypes) {
	case 0:
		return "", errors.New(fmt.Sprintf("Failed to find type of resource '%s'", resource.Name))
	case 1:
		return detectedTypes[0], nil
	default:
		return "", errors.New(fmt.Sprintf("Detected multiple types for resource '%s': %s", resource.Name, strings.Join(detectedTypes, ", ")))
	}
}

// Loops through the list of hosts in order, and attempts to fetch a
//   kubeconfig file that will allow access to the cluster.
func getKubectlFromAnyMaster(log *logrus.Entry, masters []string) (*kubeutil.Kubectl, error) {
	for _, host := range masters {
		log.Debug("Trying to fetch kubeconfig from host ", host, " from masters list")
		kubectl, err := hope.GetKubectl(host)
		if err == nil {
			return kubectl, nil
		}
	}

	return nil, errors.New("Failed to find a kubeconfig file on any host")
}

func getResources() (*[]Resource, error) {
	var resources []Resource
	err := viper.UnmarshalKey("resources", &resources)
	return &resources, err
}

func getJob(jobName string) (*Job, error) {
	var jobs []Job
	err := viper.UnmarshalKey("jobs", &jobs)
	if err != nil {
		return nil, err
	}

	for _, job := range jobs {
		if job.Name == jobName {
			return &job, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Failed to find a job named %s", jobName))
}
