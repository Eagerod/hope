package hypervisors

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/proxmox"
)

type ProxmoxHypervisor struct {
	node hope.Node

	pc *proxmox.ApiClient
}

func (p *ProxmoxHypervisor) Initialize(node hope.Node) error {
	p.node = node
	p.pc = proxmox.NewApiClient(p.node.User, p.node.Host)
	return nil
}

func (p *ProxmoxHypervisor) ListNodes() ([]string, error) {
	return p.pc.GetVmNames(p.node.Name)
}

func (p *ProxmoxHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	ip, err := p.VMIPAddress(node.Name)
	if err != nil {
		return hope.Node{}, err
	}

	node.Hypervisor = ""
	node.Host = ip
	return node, nil
}

func (p *ProxmoxHypervisor) UnderlyingNode() (hope.Node, error) {
	return p.node, nil
}

func (p *ProxmoxHypervisor) CopyImage(packer.JsonSpec, hope.VMs, hope.VMImageSpec) error {
	return fmt.Errorf("must create vm images independently on target hosts")
}

func (p *ProxmoxHypervisor) CreateImage(vms hope.VMs, vmImageSpec hope.VMImageSpec, args []string, force bool) (*packer.JsonSpec, error) {
	vmDir := path.Join(vms.Root, vmImageSpec.Name)

	log.Debugf("Copying contents of %s for parameter replacement.", vmDir)
	tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, vmImageSpec.Parameters)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	allArgs := []string{"build"}
	for _, v := range args {
		allArgs = append(allArgs, "-var", v)
	}
	allArgs = append(allArgs, ".")

	packerEnvs := map[string]string{
		"PACKER_LOG": "1",
		"PKR_VAR_proxmox_node": p.node.Name,
	}

	log.Infof("Building VM Image: %s", vmImageSpec.Name)

	if err := packer.ExecPackerWdEnv(tempDir, &packerEnvs, allArgs...); err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *ProxmoxHypervisor) CreateNode(node hope.Node, vms hope.VMs, vmImageSpec hope.VMImageSpec) error {
	err := p.pc.CreateNodeFromTemplate(p.node.Name, node.Name, vmImageSpec.Name)
	if err != nil {
		return err
	}

	// TODO: Probably only wait a few minutes tops.
	log.Infof("Waiting for vm %s to appear on node %s", node.Name, p.node.Name)
	for true {
		currentVms, err := p.ListNodes()
		if err != nil {
			return err
		}

		found := false
		for _, s := range currentVms {
			if s == node.Name {
				found = true
			}
		}

		if found {
			break
		}

		log.Debugf("Node %s not found yet. Only found: %s. Waiting 5 seconds...", node.Name, strings.Join(currentVms, ","))
		time.Sleep(5 * time.Second)
	}

	config := map[string]interface{}{}
	config["cpu"] = node.Cpu
	config["memory"] = node.Memory
	config["net[0]"] = fmt.Sprintf("bridge=%s", node.Network)

	return p.pc.ConfigureNode(p.node.Name, node.Name, config)
}

func (p *ProxmoxHypervisor) StartVM(vmName string) error {
	return p.pc.PowerOnVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) StopVM(vmName string) error {
	return p.pc.PowerOffVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) DeleteVM(vmName string) error {
	return p.pc.DeleteVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) VMIPAddress(vmName string) (string, error) {
	return p.pc.GetNodeIP(p.node.Name, vmName)
}
