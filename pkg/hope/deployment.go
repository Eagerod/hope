package hope

import (
	"github.com/Eagerod/hope/pkg/kubeutil"
)

func KubectlApplyF(kubectl *kubeutil.Kubectl, path string) error {
	return kubeutil.ExecKubectl(kubectl, "apply", "-f", path)
}

func KubectlApplyStdIn(kubectl *kubeutil.Kubectl, stdin string) error {
	return kubeutil.InKubectl(kubectl, stdin, "apply", "-f", "-")
}
