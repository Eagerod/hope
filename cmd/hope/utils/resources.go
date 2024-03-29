package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
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
			return nil, fmt.Errorf("multiple resources found in configuration file named: %s", resource.Name)
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
		return nil, fmt.Errorf("failed to find resources with names: %s", strings.Join(*maputil.MapStringBoolKeys(&nameMap), ","))
	}

	return &returnSlice, nil
}

// For each parameter from a file, load the file and populate the base64
// values of the files into the properties.
//
// Does nothing to deduplicate keys.
// All plain parameters will exist in the list before file parameters.
func FlattenParameters(directParameters, fileParameters []string) ([]string, error) {
	rv := directParameters

	for _, param := range fileParameters {
		if param == "" {
			return nil, errors.New("file parameter must be in the form PARAM=<file path>")
		}

		paramComponents := strings.SplitAfterN(param, "=", 2)

		if len(paramComponents) != 2 {
			return nil, fmt.Errorf("file parameter %s must provide file path", param)
		}

		paramName := strings.TrimRight(paramComponents[0], "=")
		paramPath := paramComponents[1]

		if paramName == "" {
			return nil, errors.New("file parameter must include a name")
		}

		if stat, err := os.Stat(paramPath); err != nil {
			return nil, err
		} else if stat.IsDir() {
			return nil, fmt.Errorf("cannot resolve parameter %s contents from directory: %s", paramName, paramPath)
		}

		contents, err := hope.ReplaceParametersInFile(paramPath, directParameters)
		if err != nil {
			return nil, err
		}

		log.Tracef("Writing base64ed contents of file %s to parameter %s", paramPath, paramName)
		b64Content := base64.StdEncoding.EncodeToString([]byte(contents))

		expandedParam := fmt.Sprintf("%s=%s", paramName, b64Content)
		rv = append(rv, expandedParam)
	}

	return rv, nil
}
