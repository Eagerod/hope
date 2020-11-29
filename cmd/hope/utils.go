package cmd

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/maputil"
)

const (
	ResourceTypeFile        string = "file"
	ResourceTypeInline      string = "inline"
	ResourceTypeDockerBuild string = "build"
	ResourceTypeJob         string = "job"
	ResourceTypeExec        string = "exec"
)

// Should be defined in hope pkg
type BuildSpec struct {
	Path   string
	Source string
	Tag    string
	Pull   string
}

type ExecSpec struct {
	Selector string
	Timeout  string
	Command  []string
}

type Resource struct {
	Name       string
	File       string
	Inline     string
	Parameters []string
	Build      BuildSpec
	Job        string
	Exec       ExecSpec
	Tags       []string
}

// TODO: Allow jobs to define max retry parameters, or accept them on the
//   command line.
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
	if (len(resource.Build.Path) != 0 || len(resource.Build.Source) != 0) && len(resource.Build.Tag) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeDockerBuild)
	}
	if len(resource.Job) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeJob)
	}
	if len(resource.Exec.Selector) != 0 && len(resource.Exec.Command) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeExec)
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

func getResources() (*[]Resource, error) {
	var resources []Resource
	err := viper.UnmarshalKey("resources", &resources)

	nameMap := map[string]bool{}
	for _, resource := range resources {
		if _, ok := nameMap[resource.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Multiple resources found in configuration file named: %s", resource.Name))
		}
		nameMap[resource.Name] = true
	}

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

func replaceParametersInString(str string, parameters []string) (string, error) {
	t := hope.NewTextSubstitutorFromString(str)
	return replaceParametersWithSubstitutor(t, parameters)
}

func replaceParametersInFile(path string, parameters []string) (string, error) {
	t, err := hope.TextSubstitutorFromFilepath(path)
	if err != nil {
		return "", err
	}

	return replaceParametersWithSubstitutor(t, parameters)
}

func replaceParametersWithSubstitutor(t *hope.TextSubstitutor, parameters []string) (string, error) {
	envParams := []string{}
	directParams := map[string]string{}
	for _, value := range parameters {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 1 {
			envParams = append(envParams, value)
		} else {
			directParams[parts[0]] = parts[1]
		}
	}

	if err := t.SubstituteTextFromEnv(envParams); err != nil {
		return "", err
	}

	if err := t.SubstituteTextFromMap(directParams); err != nil {
		return "", err
	}

	return string(*t.Bytes), nil
}

func getIdentifiableResources(names *[]string, tags *[]string) (*[]Resource, error) {
	returnSlice := []Resource{}
	nameMap := map[string]bool{}
	tagMap := map[string]bool{}

	for _, name := range *names {
		nameMap[name] = true
	}

	for _, tag := range *tags {
		tagMap[tag] = true
	}

	// Loop through all resources, adding them as they appear in the resources
	//   object in the yaml file.
	resources, err := getResources()
	if err != nil {
		return nil, err
	}

	for _, resource := range *resources {
		if _, ok := nameMap[resource.Name]; ok {
			returnSlice = append(returnSlice, resource)
			delete(nameMap, resource.Name)
			continue
		}

		for _, tag := range resource.Tags {
			if _, ok := tagMap[tag]; ok {
				returnSlice = append(returnSlice, resource)
				break
			}
		}
	}

	// If any name wasn't found, error out.
	if len(nameMap) != 0 {
		return nil, errors.New(fmt.Sprintf("Failed to find resources with names: %s", strings.Join(*maputil.MapStringBoolKeys(&nameMap), ",")))
	}

	return &returnSlice, nil
}
