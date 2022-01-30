package esxi

import (
	"encoding/json"
	"fmt"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

const VmStatePoweredOn string = "Powered on"
const VmStatePoweredOff string = "Powered off"

type vimCmdGetGuestOutput struct {
	IpAddress string `json:"ipAddress"`
}

func PowerOnVm(host string, vmId string) error {
	powerState, err := PowerStateOfVm(host, vmId)
	if err != nil {
		return err
	}

	if powerState == VmStatePoweredOn {
		return nil
	}

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
	powerState, err := PowerStateOfVm(host, vmId)
	if err != nil {
		return err
	}

	if powerState == VmStatePoweredOff {
		return nil
	}

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
	vmId, err := idFromName(host, vmName)
	if err != nil {
		return "", err
	}

	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/get.guest", vmId)
	if err != nil {
		return "", err
	}

	cleanedOutput := VimCmdParseOutput(output)

	var outputObj vimCmdGetGuestOutput
	if err := json.Unmarshal([]byte(cleanedOutput), &outputObj); err != nil {
		return "", err
	}

	// Core VM information worked.
	if outputObj.IpAddress != "" {
		return outputObj.IpAddress, nil
	}

	// Guest OS may have tools installed, fall-back to using esxcli.
	vmWorldId, err := worldIdFromName(host, vmName)
	if err != nil {
		return "", err
	}

	output, err = ssh.GetSSH(host, "esxcli", "--formatter", "csv", "--format-param", "fields=IPAddress", "network", "vm", "port", "list", "-w", vmWorldId)
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

func DeleteVm(host string, vmId string) error {
	return ssh.ExecSSH(host, "vim-cmd", "vmsvc/destroy", vmId)
}

func DeleteVmNamed(host string, vmName string) error {
	vmId, err := idFromName(host, vmName)
	if err != nil {
		return err
	}

	return DeleteVm(host, vmId)
}

func PowerStateOfVm(host string, vmId string) (string, error) {
	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/power.getstate", vmId)
	if err != nil {
		return "", err
	}

	outputLines := strings.Split(output, "\n")
	if len(outputLines) < 2 {
		return "", fmt.Errorf("Failed to parse power state from: %s", output)
	}

	switch outputLines[1] {
	case VmStatePoweredOff, VmStatePoweredOn:
		return outputLines[1], nil
	}

	return "", fmt.Errorf("Unknown power state: %s", output)
}

func PowerStateOfVmNamed(host string, vmName string) (string, error) {
	vmId, err := idFromName(host, vmName)
	if err != nil {
		return "", err
	}

	return PowerStateOfVm(host, vmId)
}
