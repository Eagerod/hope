package hypervisors

import (
	"fmt"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

func ToHypervisor(node hope.Node) (Hypervisor, error) {
	if !node.IsHypervisor() {
		return nil, fmt.Errorf("Node named %s is not a hypervisor", node.Name)
	}

	switch node.Engine {
	case "esxi":
		return &EsxiHypervisor{node}, nil
	}

	return nil, fmt.Errorf("Failed to resolve hypervisor engine: %s", node.Engine)
}
