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
	Name       string
	File       string
	Inline     string
	Parameters []string
	Build      BuildSpec
	Job        string
	Exec       ExecSpec
	Tags       []string
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
}

// VMImageSpec - Defines the structure needed to populate a Packer job to
//   build a VM Image.
type VMImageSpec struct {
	Name       string
	Parameters []string
}

// VMs - Object defining path information for building any VMs.
type VMs struct {
	Images []VMImageSpec
	Cache  string
	Output string
	Root   string
}

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

	return "UNDEFINED"
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
	return node.Role == "master+node"
}

// IsMaster - Whether or not this node is a control plane node.
func (node *Node) IsMaster() bool {
	return node.Role == "master" || node.IsMasterAndNode()
}

// IsNode - Whether or not this node is a worker node.
func (node *Node) IsNode() bool {
	return node.Role == "node" || node.IsMasterAndNode()
}

// IsHypervisor - Whether or not this node is a hypervisor node.
func (node *Node) IsHypervisor() bool {
	return node.Role == "hypervisor"
}

// IsLoadBalancer - Whether or not this node is a load-balancer node.
func (node *Node) IsLoadBalancer() bool {
	return node.Role == "load-balancer"
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
