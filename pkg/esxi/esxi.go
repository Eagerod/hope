package esxi

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func GetVmId(host string, vmName string) (string, error) {
	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/getallvms")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(output, "\n") {
		// Vmid Name File Guest_OS Version Annotation
		fields := strings.Fields(line)
		if fields[1] == vmName {
			return fields[0], nil
		}
	}

	return "", errors.New(fmt.Sprintf("Failed to vm named %s on %s", vmName, host))
}

func PowerOnVm(host string, vmId string) error {
	return ssh.ExecSSH(host, "vim-cmd", "vmsvc/power.on", vmId)
}

func PowerOnVmNamed(host string, vmName string) error {
	vmId, err := GetVmId(host, vmName)
	if err != nil {
		return err
	}

	return PowerOnVm(host, vmId)
}
