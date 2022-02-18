package hypervisors

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
)

// Hypervisor acts as a catch-all for "an entity that exposes access to manage
//   a virtual machine".
//
// ListNodes returns a list of identifiers for the nodes present on the
//   hypervisor.
//
// ResolveNode will use the contents of the provided &hope.Node, and will
//   return a new &hope.Node that can be used as though it were a physical
//   machine on the network.
//
// UnderlyingNode returns the base object used to create the hypervisor.
type Hypervisor interface {
	ListNodes() ([]string, error)
	ResolveNode(node hope.Node) (hope.Node, error)
	ValidateNodes(nodes []hope.Node) error
	UnderlyingNode() (hope.Node, error)

	CopyImage(packer.JsonSpec, hope.VMs, hope.VMImageSpec) error
	CreateImage(hope.VMs, hope.VMImageSpec, []string, bool) (*packer.JsonSpec, error)
	CreateNode(hope.Node, hope.VMs, hope.VMImageSpec) error

	StartVM(string) error
	StopVM(string) error

	// Should these interfaces also take a hope.Node, just for consistency's
	//   sake?
	DeleteVM(string) error
	VMIPAddress(string) (string, error)
}
