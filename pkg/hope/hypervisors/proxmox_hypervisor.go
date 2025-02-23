package hypervisors

import (
	"fmt"
	"os"
	"path"
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
}

func (p *ProxmoxHypervisor) ListNodes() ([]string, error) {
	return proxmox.GetNodes(p.node.User, p.node.Name, p.node.Host)
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
	}

	log.Infof("Building VM Image: %s", vmImageSpec.Name)

	if err := packer.ExecPackerWdEnv(tempDir, &packerEnvs, allArgs...); err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *ProxmoxHypervisor) CreateNode(node hope.Node, vms hope.VMs, vmImageSpec hope.VMImageSpec) error {
	err := proxmox.CreateNodeFromTemplate(p.node.User, p.node.Name, p.node.Host, node.Name, vmImageSpec.Name)
	if err != nil {
		return err
	}

	config := map[string]interface{}{}
	config["cpu"] = node.Cpu
	config["memory"] = node.Memory
	config["net[0]"] = fmt.Sprintf("bridge=", node.Network)

	return proxmox.ConfigureNode(p.node.User, p.node.Name, p.node.Host, node.Name, config)
}

func (p *ProxmoxHypervisor) StartVM(vmName string) error {
	return proxmox.PowerOnVmNamed(p.node.User, p.node.Name, p.node.Host, vmName)
}

func (p *ProxmoxHypervisor) StopVM(vmName string) error {
	return proxmox.PowerOffVmNamed(p.node.User, p.node.Name, p.node.Host, vmName)
}

func (p *ProxmoxHypervisor) DeleteVM(vmName string) error {
	return proxmox.DeleteVmNamed(p.node.User, p.node.Name, p.node.Host, vmName)
}

func (p *ProxmoxHypervisor) VMIPAddress(vmName string) (string, error) {
	return proxmox.GetNodeIP(p.node.User, p.node.Name, p.node.Host, vmName)
}
