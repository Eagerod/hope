package esxi

import (
	"errors"
	"fmt"
	"regexp"
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

func PowerOffVm(host string, vmId string) error {
	return ssh.ExecSSH(host, "vim-cmd", "vmsvc/power.off", vmId)
}

func PowerOffVmNamed(host string, vmName string) error {
	vmId, err := GetVmId(host, vmName)
	if err != nil {
		return err
	}

	return PowerOffVm(host, vmId)
}

func GetIpAddressOfVm(host string, vmId string) (string, error) {
	output, err := ssh.GetSSH(host, "vim-cmd", "vmsvc/get.guest", vmId)
	if err != nil {
		return "", err
	}

	ipAddressCleanRegexp := regexp.MustCompile("[\",\\s]")

	// Not a super easy string to parse, cause it's a PowerShell object?
	// Broken down as much as possible, and hopefully the format doesn't
	//   randomly change.
	// Search for the NIC definition, and find the IP Address attached to it.
	// If the NIC doesn't have a network set, it may just be the host network,
	//   and if that's the case, just continue on.
	lines := strings.Split(output, "\n")
	netBlockStart := 0
	for ; netBlockStart < len(lines); netBlockStart++ {
		line := lines[netBlockStart]
		if strings.HasPrefix(strings.TrimSpace(line), "net = (vim.vm.GuestInfo.NicInfo)") {
			break
		}
	}

	// Seek the (vim.vm.GuestInfo.NicInfo) block that has a network attached
	//   to it.
	for i := netBlockStart + 1; i < len(lines); i++ {
		nicInfoStartLine := lines[i]
		if strings.HasPrefix(strings.TrimSpace(nicInfoStartLine), "(vim.vm.GuestInfo.NicInfo)") {
			for j := i + 1; j < len(lines); j++ {
				line := lines[j]
				if strings.HasPrefix(strings.TrimSpace(line), "network = <unset>") {
					i = j
					break
				}

				if strings.HasPrefix(strings.TrimSpace(line), "ipAddress =") {
					ipAddressLine := lines[j+1]
					return ipAddressCleanRegexp.ReplaceAllString(ipAddressLine, ""), nil
				}
			}
		}
	}

	return "", errors.New(fmt.Sprintf("Failed to find IP Address of VM %s on %s", vmId, host))
}

func GetIpAddressOfVmNamed(host string, vmName string) (string, error) {
	vmId, err := GetVmId(host, vmName)
	if err != nil {
		return "", err
	}

	return GetIpAddressOfVm(host, vmId)
}
