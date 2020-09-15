package hope

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/scp"
)

func GetKubectl(host string) (*kubeutil.Kubectl, error) {
	remoteFile := fmt.Sprintf("%s:/etc/kubernetes/admin.conf", host)

	// Do not delete.
	// Leave deletion up to destroying the kubectl instance.
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	if err = scp.ExecSCP(remoteFile, tempFile.Name()); err != nil {
		return nil, err
	}

	kubectl := kubeutil.NewKubectl(tempFile.Name())
	return kubectl, nil
}

func FetchKubeconfig(log *logrus.Entry, host string, merge bool) error {
	kubeconfigFile, err := kubeutil.GetKubeConfigPath()
	if err != nil {
		return err
	}

	// If the file already exists, and merge isn't provided, bail.
	log.Trace("Local KUBECONFIG filepath: ", kubeconfigFile)
	if _, err := os.Stat(kubeconfigFile); err == nil {
		if !merge {
			return errors.New("Refusing to overwrite existing kubeconfig file.")
		}
	}

	kubectl, err := GetKubectl(host)
	if err != nil {
		return err
	}

	defer kubectl.Destroy()

	log.Debug("Merging existing KUBECONFIG file with file downloaded from ", host)

	combinerKubeconfig := kubeutil.NewKubectl(kubeconfigFile + ":" + kubectl.KubeconfigPath)
	kubeconfigContents, err := kubeutil.GetKubectl(combinerKubeconfig, "config", "view", "--raw")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(kubeconfigFile, []byte(kubeconfigContents), 0600)
	if err != nil {
		return err
	}

	return nil
}
