package hope

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

func KubectlApplyF(kubectl *kubeutil.Kubectl, path string) error {
	return kubeutil.ExecKubectl(kubectl, "apply", "-f", path)
}
