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

func FetchKubeconfig(log *logrus.Entry, host string, merge bool) error {
	remoteFile := fmt.Sprintf("%s:/etc/kubernetes/admin.conf", host)

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
	} else {
		log.Debug("No file found at KUBECONFIG path. Writing file directly.")
		return scp.ExecSCP(remoteFile, kubeconfigFile)
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}

	defer os.Remove(tempFile.Name())

	if err = scp.ExecSCP(remoteFile, tempFile.Name()); err != nil {
		return err
	}

	log.Debug("Merging existing KUBECONFIG file with file downloaded from ", host)

	// TODO: This _should_ be done by providing the environment to the
	//   subprocess, rather than modifying the current process' env, just so
	//   the subprocess inherits it.
	// TODO: This currently gets the combined stdout+err stream. It should only
	//   take stdout.
	kubeconfigEnv := kubeconfigFile + ":" + tempFile.Name()
	oldKubeconfigEnv := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigEnv)
	kubeconfigContents, err := kubeutil.GetKubectl("config", "view", "--raw")
	os.Setenv("KUBECONFIG", oldKubeconfigEnv)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(kubeconfigFile, []byte(kubeconfigContents), 0600)
	if err != nil {
		return err
	}

	return nil
}
