package hypervisors

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
)

type ProxmoxHypervisor struct {
	node hope.Node
}

func (p *ProxmoxHypervisor) ListNodes() ([]string, error) {
	return []string{}, nil
}

func (p *ProxmoxHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	return hope.Node{}, nil
}

func (p *ProxmoxHypervisor) UnderlyingNode() (hope.Node, error) {
	return hope.Node{}, nil
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

func (p *ProxmoxHypervisor) StartVM(string) error {
	return nil
}

func (p *ProxmoxHypervisor) StopVM(string) error {
	return nil
}

func (p *ProxmoxHypervisor) DeleteVM(string) error {
	return nil
}

func (p *ProxmoxHypervisor) VMIPAddress(string) (string, error) {
	return "", nil
}
