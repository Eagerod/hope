package hope

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
)

import (
	"github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/fileutil"
	"github.com/Eagerod/hope/pkg/kubeutil"
)

func FetchKubeconfig(log *logrus.Entry, node *Node, merge bool) error {
	kubeconfigFile, err := kubeutil.GetKubeConfigPath()
	if err != nil {
		return err
	}

	connectionString := node.ConnectionString()
	kubectl, err := kubeutil.NewKubectlFromNode(connectionString)
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
		if err := os.MkdirAll(path.Dir(kubeconfigFile), 0700); err != nil {
			return err
		}

		log.Debug("Local kubeconfig file does not exist. Writing new file.")
		if err := fileutil.CopyFileMode(kubectl.KubeconfigPath, kubeconfigFile, 0600); err != nil {
			return err
		}

		return nil
	} else {
		return err
	}

	log.Debug("Merging existing KUBECONFIG file with file downloaded from ", connectionString)

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
