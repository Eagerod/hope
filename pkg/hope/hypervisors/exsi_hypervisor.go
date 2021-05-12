package hypervisors

import (
	"fmt"
	"strings"
)

import (
	"github.com/Eagerod/hope/pkg/esxi"
	"github.com/Eagerod/hope/pkg/hope"
)

type EsxiHypervisor struct {
	node *hope.Node
}

func (hyp *EsxiHypervisor) ListNodes() (*[]string, error) {
	return esxi.ListVms(hyp.node.ConnectionString())
}

func (hyp *EsxiHypervisor) ResolveNode(node *hope.Node) (*hope.Node, error) {
	ip, err := esxi.GetIpAddressOfVmNamed(hyp.node.ConnectionString(), node.Name)
	if err != nil {
		return nil, err
	}

	ip = strings.TrimSpace(ip)
	if ip == "0.0.0.0" {
		return nil, fmt.Errorf("Failed to find IP for vm %s on %s", node.Name, hyp.node.Name)
	}

	newNode := *node
	newNode.Hypervisor = ""
	newNode.Host = ip
	return &newNode, nil
}

func (hyp *EsxiHypervisor) UnderlyingNode() (*hope.Node, error) {
	fmt.Printf("Node is: %s", hyp.node.Name)
	return hyp.node, nil
}
