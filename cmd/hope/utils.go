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
	ResourceTypeFile   string = "file"
	ResourceTypeInline string = "inline"
)

// Should be defined in hope pkg
type Resource struct {
	Name   string
	File   string
	Inline string
}

func (resource *Resource) GetType() (string, error) {
	detectedTypes := []string{}
	if len(resource.File) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeFile)
	}
	if len(resource.Inline) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeInline)
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
