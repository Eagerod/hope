package kubeutil

import (
	"errors"
	"io/ioutil"
	"os"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
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

func NewKubectlFromNode(host string) (*Kubectl, error) {
	// Do not delete.
	// Leave deletion up to destroying the kubectl instance.
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	// Because auth for the user in the config may not be for root, can't
	//   reliably pull data via scp.
	output, err := ssh.GetSSH(host, "sudo", "cat", "/etc/kubernetes/admin.conf")
	if err != nil {
		return nil, err
	}

	if _, err = tempFile.Write([]byte(output)); err != nil {
		return nil, err
	}

	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	kubectl := NewKubectl(tempFile.Name())
	return kubectl, nil
}

func (kubectl *Kubectl) Destroy() error {
	return os.Remove(kubectl.KubeconfigPath)
}
