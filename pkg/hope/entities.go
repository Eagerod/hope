package hope

import (
	"fmt"
	"strings"
)

// ResourceType enum to differentiate the types of resource definitions that
//   can appear in the hope yaml file.
type ResourceType int

const (
	// ResourceTypeUnknown - No resource type could be determined for the
	//   resource being evaluated.
	ResourceTypeUnknown ResourceType = iota

	// ResourceTypeFile - Provide a path to a local file/URL to a remote file
	//    to apply.
	ResourceTypeFile

	// ResourceTypeInline - Provide an inline yaml definition of resources to
	//   apply.
	ResourceTypeInline

	// ResourceTypeDockerBuild - Build a docker image with the given context
	//   path, and push it to the specified repository.
	ResourceTypeDockerBuild

	// ResourceTypeJob - Wait for a job with the given name to finish
	//   executing.
	ResourceTypeJob

	// ResourceTypeExec - Execute a script in a running pod/container.
	ResourceTypeExec
)

type NodeRole int

const (
	// NodeRoleHypervisor - Host that manages the other nodes listed.
	NodeRoleHypervisor NodeRole = iota

	// NodeRoleLoadBalancer - API Server load balancer VM that runs an
	//   instance of NGINX pointing at the Master nodes' API servers.
	NodeRoleLoadBalancer

	// NodeRoleMaster - Control plane nodes
	NodeRoleMaster

	// NodeRoleMasterAndNode - For small clusters, a machine that acts as both
	//   master and node.
	// Master node with the master:NoSchedule taint removed.
	NodeRoleMasterAndNode

	// NodeRoleNode - Plain Kubernetes node.
	NodeRoleNode
)

type NodeStatus int

const (
	// NodeStatusUnavailable - Something probably went wrong, and it couldn't
	//   be determined what the state of this node is.
	// Maybe the node exists, but it isn't serving properly, but in general,
	//   the node was found, but isn't but isn't doing what it's supposed to.
	NodeStatusUnavailable NodeStatus = iota

	// NodeStatusHealthy - Node is doing exactly what it should be doing.
	NodeStatusHealthy

	// NodeStatusDoesNotExist - Node is not available on Kubernetes, and not
	//   visible on its hypervisor.
	NodeStatusDoesNotExist
)

type NodeTaint struct {
	Key    string
	Value  string
	Effect string
}

// BuildSpec - Properties of a ResourceTypeDockerBuild
type BuildSpec struct {
	Path   string
	Source string
	Tag    string
	Pull   string
}

// ExecSpec - Properties of a ResourceTypeExec
type ExecSpec struct {
	Selector string
	Timeout  string
	Command  []string
}

// Resource - Properties that can appear in any resources.
// There may be a better way of doing this, but with a pretty generic list of
//   items appearing in a yaml file, maybe not.
type Resource struct {
	Name           string
	File           string
	Inline         string
	Parameters     []string
	FileParameters []string
	Build          BuildSpec
	Job            string
	Exec           ExecSpec
	Tags           []string
}

// Job - Properties that can appear in any ephemeral job definition.
// TODO: Allow jobs to define max retry parameters, or accept them on the
//   command line.
type Job struct {
	Name       string
	File       string
	Parameters []string
}

// Node - Defines a networked resource on which operations will typically be
//   executed.
// Datastore is really only used for Hypervisors, but whatever; it's not
//   incredibly intuitive how to have non-homogenous types in viper lists.
// If a more concrete type is eventually used, the Role property should become
//   an enum/bitfield.
type Node struct {
	Name       string
	Role       string
	Host       string
	Hypervisor string
	Engine     string
	User       string
	Datastore  string
	Network    string
	Cpu        int
	Memory     int
	Taints     []NodeTaint
}

// VMImageSpec - Defines the structure needed to populate a Packer job to
//   build a VM Image.
type VMImageSpec struct {
	Name        string
	Hypervisors []string
	Parameters  []string
}

// VMs - Object defining path information for building any VMs.
type VMs struct {
	Images []VMImageSpec
	Cache  string
	Output string
	Root   string
}

// Not using stringer generation because of user-provided strings.
// Not using arrays to prevent ordering issues.
func (rt ResourceType) String() string {
	switch rt {
	case ResourceTypeFile:
		return "file"
	case ResourceTypeInline:
		return "inline"
	case ResourceTypeDockerBuild:
		return "docker"
	case ResourceTypeJob:
		return "job"
	case ResourceTypeExec:
		return "exec"
	}

	return fmt.Sprintf("%%!ResourceType(%d)", rt)
}

func (nr NodeRole) String() string {
	switch nr {
	case NodeRoleHypervisor:
		return "hypervisor"
	case NodeRoleLoadBalancer:
		return "load-balancer"
	case NodeRoleMaster:
		return "master"
	case NodeRoleMasterAndNode:
		return "master+node"
	case NodeRoleNode:
		return "node"
	}

	return fmt.Sprintf("%%!NodeRole(%d)", nr)
}

func (ns NodeStatus) String() string {
	switch ns {
	case NodeStatusUnavailable:
		return "Unavailable"
	case NodeStatusHealthy:
		return "Healthy"
	case NodeStatusDoesNotExist:
		return "DoesNotExist"
	}

	return fmt.Sprintf("%%!NodeStatus(%d)", ns)
}

// GetType - Scan through defined properties, and return the resource type
//   that the resource appears to implement.
func (resource *Resource) GetType() (ResourceType, error) {
	detectedTypes := []ResourceType{}
	if len(resource.File) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeFile)
	}
	if len(resource.Inline) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeInline)
	}
	if (len(resource.Build.Path) != 0 || len(resource.Build.Source) != 0) && len(resource.Build.Tag) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeDockerBuild)
	}
	if len(resource.Job) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeJob)
	}
	if len(resource.Exec.Selector) != 0 && len(resource.Exec.Command) != 0 {
		detectedTypes = append(detectedTypes, ResourceTypeExec)
	}

	switch len(detectedTypes) {
	case 0:
		return ResourceTypeUnknown, fmt.Errorf("Failed to find type of resource '%s'", resource.Name)
	case 1:
		return detectedTypes[0], nil
	default:
		detectedTypeStrings := []string{}
		for _, i := range detectedTypes {
			detectedTypeStrings = append(detectedTypeStrings, i.String())
		}
		return ResourceTypeUnknown, fmt.Errorf("Detected multiple types for resource '%s': %s", resource.Name, strings.Join(detectedTypeStrings, ", "))
	}
}

// ConnectionString - Get the node's connection string
func (node *Node) ConnectionString() string {
	if node.User != "" {
		return fmt.Sprintf("%s@%s", node.User, node.Host)
	}

	return node.Host
}

// IsMasterAndNode - Whether or not this node plays the roles of both control
//   plane and worker node.
func (node *Node) IsMasterAndNode() bool {
	return node.Role == NodeRoleMasterAndNode.String()
}

// IsMaster - Whether or not this node is a control plane node.
func (node *Node) IsMaster() bool {
	return node.Role == NodeRoleMaster.String() || node.IsMasterAndNode()
}

// IsNode - Whether or not this node is a worker node.
func (node *Node) IsNode() bool {
	return node.Role == NodeRoleNode.String() || node.IsMasterAndNode()
}

// IsHypervisor - Whether or not this node is a hypervisor node.
func (node *Node) IsHypervisor() bool {
	return node.Role == NodeRoleHypervisor.String()
}

// IsLoadBalancer - Whether or not this node is a load-balancer node.
func (node *Node) IsLoadBalancer() bool {
	return node.Role == NodeRoleLoadBalancer.String()
}

// IsKubernetesNode - Whether or not this node has one of the Kubernetes
//   roles.
func (node *Node) IsKubernetesNode() bool {
	return node.IsMaster() || node.IsNode()
}

// IsRoleValid - Whether or not the node has a role that has been implemented.
func (node *Node) IsRoleValid() bool {
	return node.IsKubernetesNode() || node.IsHypervisor() || node.IsLoadBalancer()
}
