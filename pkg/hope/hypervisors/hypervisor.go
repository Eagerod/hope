package hypervisors

import (
	"errors"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
)

var ErrCopyImageNotImplemented error = errors.New("CopyImage not implemented for this hypervisor")

type CopyImageMode int

const (
	// The hypervisor does not support copying images between instances.
	// Invocations to `CopyImage` should result in a
	// `ErrCopyImageNotImplemented`
	CopyImageModeNone CopyImageMode = iota

	// After calling `CreateImage`, the user can reliably invoke `CopyImage`
	// for each hypervisor in the hypervisor list.
	CopyImageModeToAll

	// After calling `CreateImage`, the user can reliably invoke `CopyImage`
	// for each hypervisor _except_ the one with which `CreateImage`` was
	// invoked.
	CopyImageModeFromFirst
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

	// How instances this hypervisor expect images to be copied
	CopyImageMode() CopyImageMode

	// Copy an image from the packer cache to all hypervisors it should exist
	// on.
	CopyImage(hope.VMs, hope.VMImageSpec, Hypervisor) error

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
