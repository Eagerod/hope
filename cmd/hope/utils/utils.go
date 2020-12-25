// Package utils -- Utility functions to be used throughout cmd.
// Kind of decided that this was a pretty decent way of doing shared logic for
//   cmd.
// Basically the same as for pkg, so nothing too special there, but since cmd
//   has a bit of nesting, it brings up some more questions.
// It seems like golang itself does a pattern like this for `base` cmd units,
//   so this is probably not too shabby.
package utils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/fileutil"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

func ReplaceParametersInString(str string, parameters []string) (string, error) {
	t := hope.NewTextSubstitutorFromString(str)
	return ReplaceParametersWithSubstitutor(t, parameters)
}

func ReplaceParametersInFile(path string, parameters []string) (string, error) {
	t, err := hope.TextSubstitutorFromFilepath(path)
	if err != nil {
		return "", err
	}

	return ReplaceParametersWithSubstitutor(t, parameters)
}

func ReplaceParametersWithSubstitutor(t *hope.TextSubstitutor, parameters []string) (string, error) {
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

func replaceParametersInDirectory(dir string, parameters []string) error {
	return filepath.Walk(dir, func(apath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		str, err := ReplaceParametersInFile(apath, parameters)
		if err != nil {
			return err
		}

		return fileutil.WriteFile(str, apath)
	})
}

// ReplaceParametersInDirectoryCopy - Copy the provided directory, and replace
//   parameters in the files.
// Returns the temp path to the copied directory, and the caller must clean up
//   that directory itself, unless an error occurs.
func ReplaceParametersInDirectoryCopy(dir string, parameters []string) (string, error) {
	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		return "", err
	}

	if err := fileutil.CopyDirectory(dir, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	if len(parameters) != 0 {
		if err := replaceParametersInDirectory(tempDir, parameters); err != nil {
			os.RemoveAll(tempDir)
			return "", err
		}
	}

	return tempDir, nil
}

// Loops through the list of hosts in order, and attempts to fetch a
//   kubeconfig file that will allow access to the cluster.
func GetKubectlFromAnyMaster(log *logrus.Entry, masters []string) (*kubeutil.Kubectl, error) {
	for _, host := range masters {
		log.Debug("Trying to fetch kubeconfig from host ", host, " from masters list")
		kubectl, err := kubeutil.NewKubectlFromNode(host)
		if err == nil {
			return kubectl, nil
		}
	}

	return nil, errors.New("Failed to find a kubeconfig file on any host")
}
