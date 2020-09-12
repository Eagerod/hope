package kubeutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

import (
	homedir "github.com/mitchellh/go-homedir"
)

type GetKubeutilFunc func(args ...string) (string, error)
type ExecKubeutilFunc func(args ...string) error

var GetKubectl GetKubeutilFunc = func(args ...string) (string, error) {
	osCmd := exec.Command("kubectl", args...)
	osCmd.Stdin = os.Stdin
	output, err := osCmd.CombinedOutput()

	return string(output), err
}

var ExecKubectl ExecKubeutilFunc = func(args ...string) error {
	osCmd := exec.Command("kubectl", args...)
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
}

// Get the name by which the cluster recognizes a given host.
func NodeNameFromHost(host string) (string, error) {
	nodesOutput, err := GetKubectl("get", "nodes", "-o", "custom-columns=NODE:metadata.name,IP:status.addresses[?(@.type=='InternalIP')].address")
	if err != nil {
		return "", errors.New(strings.Join([]string{nodesOutput, err.Error()}, " "))
	}

	outputRows := strings.Split(nodesOutput, "\n")
	if len(outputRows) < 2 {
		return "", errors.New("No nodes found in this cluster")
	}

	nodeRows := outputRows[1:]

	for _, nodeRow := range nodeRows {
		if strings.HasPrefix(nodeRow, host) || strings.HasSuffix(nodeRow, host) {
			return strings.Split(nodeRow, " ")[0], nil
		}
	}

	return "", errors.New(fmt.Sprintf("Host: %s not found in this cluster", host))
}

func GetKubeConfigPath() (string, error) {
	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if kubeconfigEnv != "" {
		return kubeconfigEnv, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return path.Join(home, ".kube", "config"), nil
}
