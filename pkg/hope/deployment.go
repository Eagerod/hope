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

func KubectlCreateStdIn(kubectl *kubeutil.Kubectl, stdin string) error {
	return kubeutil.InKubectl(kubectl, stdin, "create", "-f", "-")
}

func KubectlGetCreateStdIn(kubectl *kubeutil.Kubectl, stdin string) (string, error) {
	return kubeutil.GetInKubectl(kubectl, stdin, "create", "-f", "-")
}

func KubectlDeleteF(kubectl *kubeutil.Kubectl, path string) error {
	return kubeutil.ExecKubectl(kubectl, "delete", "--ignore-not-found", "-f", path)
}

func KubectlDeleteStdIn(kubectl *kubeutil.Kubectl, stdin string) error {
	return kubeutil.InKubectl(kubectl, stdin, "delete", "--ignore-not-found", "-f", "-")
}
