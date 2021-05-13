package hypervisors

import (
	"fmt"
	"strings"

	"github.com/Eagerod/hope/pkg/esxi"
	"github.com/Eagerod/hope/pkg/hope"
)

type EsxiHypervisor struct {
	node hope.Node
}

func (hyp EsxiHypervisor) ListNodes() ([]string, error) {
	v, e := esxi.ListVms(hyp.node.ConnectionString())
	if e == nil {
		return *v, nil
	}
	return nil, e
}

func (hyp *EsxiHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	ip, err := esxi.GetIpAddressOfVmNamed(hyp.node.ConnectionString(), node.Name)
	if err != nil {
		return hope.Node{}, err
	}

	ip = strings.TrimSpace(ip)
	if ip == "0.0.0.0" {
		return hope.Node{}, fmt.Errorf("Failed to find IP for vm %s on %s", node.Name, hyp.node.Name)
	}

	node.Hypervisor = ""
	node.Host = ip
	return node, nil
}

func (hyp *EsxiHypervisor) UnderlyingNode() (hope.Node, error) {
	return hyp.node, nil
}
