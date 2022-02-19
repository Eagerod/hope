package node

import (
	"errors"
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/sliceutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

var statusCmdTypeSlice *[]string

func initStatusCmd() {
	statusCmdTypeSlice = statusCmd.Flags().StringArrayP("type", "t", []string{}, "validate nodes of this type")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks every node in the manifest, and prints its name, along with its status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(*statusCmdTypeSlice) == 0 {
			*statusCmdTypeSlice = []string{
				hope.NodeRoleHypervisor.String(),
				hope.NodeRoleLoadBalancer.String(),
				hope.NodeRoleMaster.String(),
				hope.NodeRoleMasterAndNode.String(),
				hope.NodeRoleNode.String(),
			}
		}

		nodes, err := utils.GetBareNodeTypes(*statusCmdTypeSlice)
		if err != nil {
			return err
		}

		// If there are any Kubernetes nodes to be checked, cache a kubectl
		//   instance.
		var kubectl *kubeutil.Kubectl
		defer func() {
			if kubectl != nil {
				kubectl.Destroy()
			}
		}()
		if sliceutil.StringInSlice(hope.NodeRoleMaster.String(), *statusCmdTypeSlice) ||
			sliceutil.StringInSlice(hope.NodeRoleNode.String(), *statusCmdTypeSlice) ||
			sliceutil.StringInSlice(hope.NodeRoleMasterAndNode.String(), *statusCmdTypeSlice) {
				var err error
				kubectl, err = utils.KubectlFromAnyMaster()
				if err != nil {
					return err
				}
		}

		nodeStatuses := map[string]hope.NodeStatus{}
		shouldFail := false
		for _, node := range nodes {
			nodeStatuses[node.Name] = hope.NodeStatusUnavailable

			switch node.Role {
			case hope.NodeRoleLoadBalancer.String():
				var status hope.NodeStatus
				status, err = loadBalancerNodeStatus(node)
				if err != nil {
					return err
				}

				nodeStatuses[node.Name] = status
			case hope.NodeRoleMaster.String(),
					hope.NodeRoleMasterAndNode.String(),
					hope.NodeRoleNode.String():
				var status hope.NodeStatus
				status, err = kubernetesNodeStatus(kubectl, node)
				if err != nil {
					return err
				}

				nodeStatuses[node.Name] = status
			case hope.NodeRoleHypervisor.String():
				var status hope.NodeStatus
				status, err = hypervisorNodeStatus(node)
				if err != nil {
					return err
				}

				nodeStatuses[node.Name] = status
			default:
				return fmt.Errorf("unknown node type: %s", node.Role)
			}

			if nodeStatuses[node.Name] != hope.NodeStatusHealthy {
				shouldFail = true
			}
		}

		for k, v := range nodeStatuses {
			fmt.Printf("%s %s\n", k, v)
		}

		if shouldFail {
			return errors.New("error with nodes; see output for more details")
		}

		return nil
	},
}

func kubernetesNodeStatus(kubectl *kubeutil.Kubectl, node hope.Node) (hope.NodeStatus, error) { 
	// GetKubectl and ignore output to avoid output to console.
	_, err := kubeutil.GetKubectl(kubectl, "get", "node", node.Name)
	if err == nil {
		return hope.NodeStatusHealthy, nil
	}

	hypervisor, err := utils.GetHypervisor(node.Hypervisor)
	if err != nil {
		return hope.NodeStatusUnavailable, err
	}

	_, err = hypervisor.VMIPAddress(node.Name)
	if err != nil {
		return hope.NodeStatusDoesNotExist, nil
	}

	log.Debugf("VM %s exists with an IP address, but isn't a Kubernetes node.", node.Name)
	return hope.NodeStatusUnavailable, nil	
}

func loadBalancerNodeStatus(node hope.Node) (hope.NodeStatus, error) {
	hypervisor, err := utils.GetHypervisor(node.Hypervisor)
	if err != nil {
		return hope.NodeStatusUnavailable, err
	}

	resolvedNode, err := hypervisor.ResolveNode(node)
	if err != nil {
		return hope.NodeStatusDoesNotExist, nil
	} else {
		cmd := []string{
			resolvedNode.ConnectionString(),
			"sudo",
			"docker",
			"ps",
			"--filter",
			"publish=6443",
			"--quiet",
		}
		output, err := ssh.GetSSH(cmd...)
		if err != nil || output == "" {
			return hope.NodeStatusUnavailable, nil
		} else {
			return hope.NodeStatusHealthy, nil
		}
	}
}

func hypervisorNodeStatus(node hope.Node) (hope.NodeStatus, error) {
	err := ssh.ExecSSH(node.ConnectionString(), "exit")
	if err != nil {
		return hope.NodeStatusUnavailable, nil
	}

	return hope.NodeStatusHealthy, nil
}
