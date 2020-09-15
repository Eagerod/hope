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

// Kubectl struct allows for execution of a kubectl command with a
//   non-environment set kubeconfig path.
type Kubectl struct {
	KubeconfigPath string
}

func NewKubectl(kubeconfigPath string) *Kubectl {
	return &Kubectl{kubeconfigPath}
}

func (kubectl *Kubectl) Destroy() error {
	return os.Remove(kubectl.KubeconfigPath)
}

type GetKubectlFunc func(kubectl *Kubectl, args ...string) (string, error)
type ExecKubectlFunc func(kubectl *Kubectl, args ...string) error

var GetKubectl GetKubectlFunc = func(kubectl *Kubectl, args ...string) (string, error) {
	osCmd := exec.Command("kubectl", args...)
    osCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubectl.KubeconfigPath))
	osCmd.Stdin = os.Stdin
	osCmd.Stdin = os.Stdin
	osCmd.Stderr = os.Stderr

	outputBytes, err := osCmd.Output()
	return string(outputBytes), err
}

var ExecKubectl ExecKubectlFunc = func(kubectl *Kubectl, args ...string) error {
	osCmd := exec.Command("kubectl", args...)
    osCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubectl.KubeconfigPath))
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
