package kubeutil

import (
	"fmt"
	"os"
	"os/exec"
	"path"
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
	output, err := osCmd.CombinedOutput()

	return string(output), err
}

var ExecKubectl ExecKubectlFunc = func(kubectl *Kubectl, args ...string) error {
	osCmd := exec.Command("kubectl", args...)
    osCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubectl.KubeconfigPath))
	osCmd.Stdin = os.Stdin
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr

	return osCmd.Run()
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
