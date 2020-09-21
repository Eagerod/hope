package kubeutil

import (
	"os"
)

// Kubectl struct allows for execution of a kubectl command with a
//   non-environment set kubeconfig path.
// TODO: Move more uses of kubeutil.ExecKubectl/GetKubectl/etc to use this
//   structure.
type Kubectl struct {
	KubeconfigPath string
}

func NewKubectl(kubeconfigPath string) *Kubectl {
	return &Kubectl{kubeconfigPath}
}

func (kubectl *Kubectl) Destroy() error {
	return os.Remove(kubectl.KubeconfigPath)
}
