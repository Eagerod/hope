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

func getResources() (*[]hope.Resource, error) {
	var resources []hope.Resource
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

func getJobs() (*[]hope.Job, error) {
	var jobs []hope.Job
	err := viper.UnmarshalKey("jobs", &jobs)

	nameMap := map[string]bool{}
	for _, job := range jobs {
		if _, ok := nameMap[job.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Multiple jobs found in configuration file named: %s", job.Name))
		}
		nameMap[job.Name] = true
	}

	return &jobs, err
}

func getJob(jobName string) (*hope.Job, error) {
	jobs, err := getJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range *jobs {
		if job.Name == jobName {
			return &job, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("Failed to find a job named %s", jobName))
}

func getNodes() (*[]hope.Node, error) {
	var nodes []hope.Node
	err := viper.UnmarshalKey("nodes", &nodes)

	nameMap := map[string]bool{}
	for _, node := range nodes {
		if _, ok := nameMap[node.Name]; ok {
			return nil, errors.New(fmt.Sprintf("Multiple nodes found in configuration file named: %s", node.Name))
		}
		nameMap[node.Name] = true
	}

	return &nodes, err
}

func getNode(name string) (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.Name == name {
			return &node, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("Failed to find a node named %s", name))
}

func getAnyMaster() (*hope.Node, error) {
	nodes, err := getNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range *nodes {
		if node.IsMaster() {
			return &node, nil
		}
	}

	return nil, errors.New("Failed to find any master in nodes config")
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

func getIdentifiableResources(names *[]string, tags *[]string) (*[]hope.Resource, error) {
	returnSlice := []hope.Resource{}
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
