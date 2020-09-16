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
	"github.com/Eagerod/hope/pkg/fileutil"
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

	// Close the file immediately, because it will be written by a subprocess.
	if err := tempFile.Close(); err != nil {
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

	kubectl, err := GetKubectl(host)
	if err != nil {
		return err
	}

	defer kubectl.Destroy()

	// If the file already exists, and merge isn't provided, bail.
	// TODO: Test if local kubeconfig path is actually pointing at multiple
	//   files, and fail gracefully for that.
	log.Trace("Local KUBECONFIG filepath: ", kubeconfigFile)
	if _, err := os.Stat(kubeconfigFile); err == nil {
		if same, _ := fileutil.FilesIdentical(kubeconfigFile, kubectl.KubeconfigPath); same {
			log.Info("File pulled from remote identical to local file. Skipping overwrite.")
			return nil
		}

		if !merge {
			return errors.New("Refusing to overwrite existing kubeconfig file.")
		}
	} else if os.IsNotExist(err) {
		log.Debug("Local kubeconfig file does not exist. Writing new file.")
		if err := fileutil.CopyFileMode(kubectl.KubeconfigPath, kubeconfigFile, 0600); err != nil {
			return err
		}

		return nil
	} else {
		return err
	}

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
