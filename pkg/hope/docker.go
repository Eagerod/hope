package hope

import (
	"fmt"
	"strings"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/docker"
)

func DoDockerBuild(log *logrus.Entry, resource Resource) error {
	isCacheCommand := len(resource.Build.Source) != 0
	isBuildCommand := len(resource.Build.Path) != 0

	if isCacheCommand && isBuildCommand {
		return fmt.Errorf("docker build step %s cannot have a path and a source", resource.Name)
	}

	// TODO: Move these to constants somewhere
	pullConstraintAlways := resource.Build.Pull == "always"
	pullConstraintIfNotPresent := resource.Build.Pull == "if-not-present" || resource.Build.Pull == ""

	if !pullConstraintAlways && !pullConstraintIfNotPresent {
		return fmt.Errorf("unknown Docker image pull constraint: %s", resource.Build.Pull)
	}

	pullImage := ""
	if isCacheCommand {
		pullImage = resource.Build.Source
	} else {
		pullImage = resource.Build.Tag
	}

	ifNotPresentShouldPull := false
	if pullConstraintIfNotPresent {
		output, err := docker.GetDocker("images", pullImage, "--format={{.Repository}}:{{.Tag}}")
		if err != nil {
			return err
		}

		outputLines := strings.Split(output, "\n")
		if len(outputLines) == 0 {
			log.Infof("No Docker images like %s not found locally, must pull from upstream.", pullImage)
			ifNotPresentShouldPull = true
		} else {
			// Figure out if the latest tag needs to be defaulted
			//   to, or if a specific one was requested.
			searchTag := pullImage
			tagIndex := strings.LastIndex(searchTag, ":")
			if tagIndex == -1 {
				log.Debug("Provided image isn't tagged; assuming latest")
				searchTag = fmt.Sprintf("%s:latest", searchTag)
			}

			log.Tracef("Searching for local copy of tag: %s", searchTag)

			imageFound := false
			for _, imageTag := range outputLines {
				if imageTag == searchTag {
					log.Debugf("Docker image matching %s found, skipping upstream pull", searchTag)
					imageFound = true
					break
				}
			}

			if !imageFound {
				log.Infof("Docker image %s not found among candidates, must pull from upstream", searchTag)
				ifNotPresentShouldPull = true
			}
		}
	}

	if ifNotPresentShouldPull || pullConstraintAlways {
		if err := docker.ExecDocker("pull", pullImage); err != nil {
			return fmt.Errorf("failed to find image named %s", pullImage)
		}
	}

	if isBuildCommand {
		if err := docker.ExecDocker("build", resource.Build.Path, "-t", resource.Build.Tag); err != nil {
			return err
		}
	} else {
		if err := docker.ExecDocker("tag", resource.Build.Source, resource.Build.Tag); err != nil {
			return err
		}
	}

	if err := docker.ExecDocker("push", resource.Build.Tag); err != nil {
		return err
	}

	return nil
}
