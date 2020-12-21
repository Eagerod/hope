package esxi

import (
	"fmt"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

func PowerOnVm(host string, vmId string) error {
	return ssh.ExecSSH(host, "vim-cmd", "vmsvc/power.on", vmId)
}

func PowerOnVmNamed(host string, vmName string) error {
	vmId, err := idFromName(host, vmName)
	if err != nil {
		return err
	}

	return PowerOnVm(host, vmId)
}

func PowerOffVm(host string, vmId string) error {
	return ssh.ExecSSH(host, "vim-cmd", "vmsvc/power.off", vmId)
}

func PowerOffVmNamed(host string, vmName string) error {
	vmId, err := idFromName(host, vmName)
	if err != nil {
		return err
	}

	return PowerOffVm(host, vmId)
}

func GetIpAddressOfVmNamed(host string, vmName string) (string, error) {
	vmWorldId, err := worldIdFromName(host, vmName)
	if err != nil {
		return "", err
	}

	output, err := ssh.GetSSH(host, "esxcli", "--formatter", "csv", "--format-param", "fields=IPAddress", "network", "vm", "port", "list", "-w", vmWorldId)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")

	// "Couldn't find VM with given world ID"
	if len(lines) == 1 {
		return "", fmt.Errorf("Failed to find IP Address of VM %s on %s", vmName, host)
	}

	ip := strings.TrimSpace(strings.Split(lines[1], ",")[0])
	return ip, nil
}

func ListVms(host string) (*[]string, error) {
	retVal := []string{}

	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/getallvms")
	if err != nil {
		return nil, err
	}

	// Line 0 is headers, so skip that.
	for _, line := range strings.Split(strings.TrimSpace(output), "\n")[1:] {
		// Vmid Name File Guest_OS Version Annotation
		fields := strings.Fields(line)
		retVal = append(retVal, fields[1])
	}
	
	return &retVal, nil
}
