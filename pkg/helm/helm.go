package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ExecHelmFunc func(args ...string) error
type GetHelmFunc func(args ...string) (string, error)

var ExecHelm ExecHelmFunc = func(args ...string) error {
	osCmd := exec.Command("helm", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

var GetHelm GetHelmFunc = func(args ...string) (string, error) {
	osCmd := exec.Command("helm", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

func HasRepo(repo, aUrl string) (bool, error) {
	currentRepos, err := GetHelm("repo", "list")
	if err != nil {
		return false, err
	}

	normalizedUrl := strings.TrimRight(aUrl, "/")

	for _, repoLine := range strings.Split(currentRepos, "\n") {
		repoComponents := strings.Fields(repoLine)
		if repoComponents[0] != repo {
			continue
		}

		repoComponents[1] = strings.TrimRight(repoComponents[1], "/")
		if repoComponents[1] != normalizedUrl {
			return false, fmt.Errorf("local helm repo '%s' has a different url than expected", repo)
		}

		return true, nil
	}

	return false, nil
}
