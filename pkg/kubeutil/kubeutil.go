package kubeutil

import (
	"os"
	"path"
)

import (
	homedir "github.com/mitchellh/go-homedir"
)

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
