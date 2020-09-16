package cmd

import (
	"errors"
)

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

type Resource struct {
	Name string
	File string
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
