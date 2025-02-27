package hypervisors

import (
	"github.com/Eagerod/hope/pkg/hope"
)

// Hypervisor acts as a catch-all for "an entity that exposes access to manage
// a virtual machine".
type Hypervisor interface {
	// Initialize using the provided Node.
	Initialize(hope.Node) error

	// Return a list of identifiers for the nodes present on the hypervisor.
	ListNodes() ([]string, error)

	// Ask the hypervisor for the host of the node, and return a new node with
	// reachable IP in its host field.
	ResolveNode(node hope.Node) (hope.Node, error)

	// Returns the base object used to create the hypervisor.
	UnderlyingNode() (hope.Node, error)

	// Create an image using the given image spec.
	CreateImage(hope.VMs, hope.VMImageSpec, []string, bool) error

	// Create a node from the given image spec.
	CreateNode(hope.Node, hope.VMs, hope.VMImageSpec) error

	// Start the VM identified by the given value.
	StartVM(string) error

	// Start the VM identified by the given value.
	StopVM(string) error

	// Delete the VM identified by the given value.
	DeleteVM(string) error

	// Get the IP address of the VM identified by the given value.
	VMIPAddress(string) (string, error)
}
