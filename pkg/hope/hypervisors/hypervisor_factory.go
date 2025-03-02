package hypervisors

import (
	"fmt"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

type ToHypervisorFactoryFunc func(hope.Node) (Hypervisor, error)

var ToHypervisor ToHypervisorFactoryFunc = func(node hope.Node) (Hypervisor, error) {
	if !node.IsHypervisor() {
		return nil, fmt.Errorf("node named %s is not a hypervisor", node.Name)
	}

	var rv Hypervisor = nil
	switch node.Engine {
	case "esxi":
		rv = &EsxiHypervisor{}
	case "proxmox":
		rv = &ProxmoxHypervisor{}
	default:
		return nil, fmt.Errorf("failed to resolve hypervisor engine: %s", node.Engine)
	}

	if err := rv.Initialize(node); err != nil {
		return nil, err
	}
	return rv, nil
}
