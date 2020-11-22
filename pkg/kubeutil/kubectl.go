package kubeutil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

import (
	"github.com/Eagerod/hope/pkg/scp"
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

func NewKubectlFromNode(host string) (*Kubectl, error) {
	remoteFile := fmt.Sprintf("%s:/etc/kubernetes/admin.conf", host)

	// Do not delete.
	// Leave deletion up to destroying the kubectl instance.
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	// Close the file immediately, because it will be written by a subprocess.
	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	if err = scp.ExecSCP(remoteFile, tempFile.Name()); err != nil {
		return nil, err
	}

	kubectl := NewKubectl(tempFile.Name())
	return kubectl, nil
}

func NewKubectlFromAnyNode(hosts []string) (*Kubectl, error) {
	allErrorsStr := ""

	for _, host := range hosts {
		kubectl, err := NewKubectlFromNode(host)
		if err == nil {
			return kubectl, nil
		}

		allErrorsStr += "  " + err.Error() + "\n"
	}

	return nil, errors.New("Failed to find a kubeconfig file on any host:\n" + allErrorsStr)
}
