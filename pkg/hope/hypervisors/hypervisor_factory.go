package hypervisors

import (
	"fmt"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

func ToHypervisor(node hope.Node) (Hypervisor, error) {
	if !node.IsHypervisor() {
		return nil, fmt.Errorf("node named %s is not a hypervisor", node.Name)
	}

	var rv Hypervisor = nil
	switch node.Engine {
	case "esxi":
		rv = &EsxiHypervisor{}
	default:
		return nil, fmt.Errorf("failed to resolve hypervisor engine: %s", node.Engine)
	}

	rv.Initialize(node)
	return rv, nil
}
