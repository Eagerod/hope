package hypervisors

import (
	"github.com/Eagerod/hope/pkg/hope"
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
type Hypervisor interface {
	ListNodes() (*[]string, error)
	ResolveNode(node *hope.Node) (*hope.Node, error)
}
