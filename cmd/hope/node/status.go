package node

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/hope/cmd/hope/utils"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/ssh"
)

var statusCmdTypeSlice *[]string

func initStatusCmd() {
	statusCmdTypeSlice = statusCmd.Flags().StringArrayP("type", "t", []string{}, "fetch status of nodes of this type")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of nodes from the manifest, and prints names, along with statuses.",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var nodes []hope.Node
		var err error
		if len(args) == 1 {
			if len(*statusCmdTypeSlice) != 0 {
				return fmt.Errorf("cannot provide a node and a type")
			}

			nodeName := args[0]
			node, err := utils.GetBareNode(nodeName)
			if err != nil {
				return err
			}

			nodes = []hope.Node{node}
		} else {
			if len(*statusCmdTypeSlice) == 0 {
				*statusCmdTypeSlice = []string{
					hope.NodeRoleHypervisor.String(),
					hope.NodeRoleLoadBalancer.String(),
					hope.NodeRoleMaster.String(),
					hope.NodeRoleMasterAndNode.String(),
					hope.NodeRoleNode.String(),
				}
			}

			nodes, err = utils.GetBareNodeTypes(*statusCmdTypeSlice)
			if err != nil {
				return err
			}
		}

		// If there are any Kubernetes nodes to be checked, cache a kubectl
		//   instance.
		var kubectl *kubeutil.Kubectl
		defer func() {
			if kubectl != nil {
				kubectl.Destroy()
			}
		}()

		for _, node := range nodes {
			if node.IsKubernetesNode() {
				kubectl, err = utils.KubectlFromAnyMaster()
				if err != nil {
					return err
				}
				break
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

		// Order output by the order in the yaml file, rather than iterating
		//   over the map.
		writer := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
		fmt.Fprintln(writer, "Node\tStatus\t")
		for _, node := range nodes {
			fmt.Fprintf(writer, "%s\t%s\t\n", node.Name, nodeStatuses[node.Name])
		}
		writer.Flush()

		if shouldFail {
			return errors.New("error with nodes; see output for more details")
		}

		return nil
	},
}

func kubernetesNodeStatus(kubectl *kubeutil.Kubectl, node hope.Node) (hope.NodeStatus, error) {
	status, err := kubeutil.GetKubectl(
		kubectl, "get", "node", node.Name,
		"-o", "template={{range .status.conditions}}{{if eq .reason \"KubeletReady\"}}{{.status}}{{end}}{{end}}")
	if err == nil && status == "True" {
		return hope.NodeStatusHealthy, nil
	} else if err == nil {
		return hope.NodeStatusUnavailable, nil
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
