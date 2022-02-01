package esxi

import (
	"fmt"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func idFromName(host string, vmName string) (string, error) {
	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/getallvms")
	if err != nil {
		return "", err
	}

	// Output has a trailing newline, so ditch that before iterating
	output = strings.TrimSpace(output)

	for _, line := range strings.Split(output, "\n") {
		// Vmid Name File Guest_OS Version Annotation
		fields := strings.Fields(line)
		if fields[1] == vmName {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("Failed to vm named %s on %s", vmName, host)
}

func worldIdFromName(host string, vmName string) (string, error) {
	output, err := ssh.GetSSH(host, "esxcli", "--formatter", "csv", "--format-param", "fields=DisplayName,WorldID", "vm", "process", "list")
	if err != nil {
		return "", err
	}

	// Super primitive CSV parsing; should be fine
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if strings.TrimSpace(fields[0]) == vmName {
			return strings.TrimSpace(fields[1]), nil
		}
	}

	return "", fmt.Errorf("Failed to find a VM named %s on %s", vmName, host)
}
