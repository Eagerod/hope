package kubeutil

import (
	"os"
	"os/exec"
	"path"
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
