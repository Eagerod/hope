package utils

import (
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

func GetResources() (*[]hope.Resource, error) {
	var resources []hope.Resource
	err := viper.UnmarshalKey("resources", &resources)

	nameMap := map[string]bool{}
	for _, resource := range resources {
		if _, ok := nameMap[resource.Name]; ok {
			return nil, fmt.Errorf("Multiple resources found in configuration file named: %s", resource.Name)
		}
		nameMap[resource.Name] = true
	}

	return &resources, err
}

func GetIdentifiableResources(names *[]string, tags *[]string) (*[]hope.Resource, error) {
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
	resources, err := GetResources()
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
		return nil, fmt.Errorf("Failed to find resources with names: %s", strings.Join(*maputil.MapStringBoolKeys(&nameMap), ","))
	}

	return &returnSlice, nil
}
