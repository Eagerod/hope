package hypervisors

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
	return hope.Node{}, nil
}

func (p *ProxmoxHypervisor) UnderlyingNode() (hope.Node, error) {
	return p.node, nil
}

func (p *ProxmoxHypervisor) CopyImage(packer.JsonSpec, hope.VMs, hope.VMImageSpec) error {
	return nil
}

func (p *ProxmoxHypervisor) CreateImage(hope.VMs, hope.VMImageSpec, []string, bool) (*packer.JsonSpec, error) {
	return nil, nil
}

func (p *ProxmoxHypervisor) CreateNode(hope.Node, hope.VMs, hope.VMImageSpec) error {
	return nil
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
