package hypervisors

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"time"
)

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/proxmox"
)

type ProxmoxHypervisor struct {
	node hope.Node

	pc *proxmox.ApiClient
}

func (p *ProxmoxHypervisor) Initialize(node hope.Node) error {
	p.node = node
	p.pc = proxmox.NewApiClient(p.node.User, p.node.Host)
	return nil
}

func (p *ProxmoxHypervisor) ListNodes() ([]string, error) {
	return p.pc.GetVmNames(p.node.Name)
}

func (p *ProxmoxHypervisor) ListBuiltImages(vms hope.VMs) ([]string, error) {
	// With the right adaptation of arguments, it might be possible to ask
	// other ProxmoxHypervisors for their list of built images.
	// With that, this client could return some union of all available
	// templates, rather than just the node attached to this instance.
	return p.pc.GetTemplateNames(p.node.Name)
}

func (p *ProxmoxHypervisor) ListAvailableImages(hope.VMs) ([]string, error) {
	return p.pc.GetTemplateNames(p.node.Name)
}

func (p *ProxmoxHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	ip, err := p.VMIPAddress(node.Name)
	if err != nil {
		return hope.Node{}, err
	}

	node.Hypervisor = ""
	node.Host = ip
	return node, nil
}

func (p *ProxmoxHypervisor) UnderlyingNode() (hope.Node, error) {
	return p.node, nil
}

func (p *ProxmoxHypervisor) CopyImageMode() CopyImageMode {
	return CopyImageModeFromFirst
}

func (p *ProxmoxHypervisor) CopyImage(vms hope.VMs, vmImageSpec hope.VMImageSpec, originalHV Hypervisor) error {
	originalProxmoxHypervisor, ok := originalHV.(*ProxmoxHypervisor)
	if !ok {
		originalNode, err := originalHV.UnderlyingNode()
		if err != nil {
			return fmt.Errorf("failed to get node from non-proxmox hypervisor in image copy")
		}
		return fmt.Errorf("failed to get node from non-proxmox hypervisor %s in image copy", originalNode.Name)
	}

	tempNodeName := uuid.New().String()
	if err := p.pc.CreateNodeFromTemplate(originalProxmoxHypervisor.node.Name, tempNodeName, vmImageSpec.Name); err != nil {
		return err
	}

	if err := originalProxmoxHypervisor.waitForNode(5*time.Second, 5*time.Minute, tempNodeName); err != nil {
		return err
	}

	// Make sure the template is actually being treated like a template.
	config := map[string]interface{}{}
	config["template"] = 1
	if err := originalProxmoxHypervisor.pc.ConfigureNode(originalProxmoxHypervisor.node.Name, tempNodeName, config); err != nil {
		return err
	}

	if err := p.pc.MigrateTemplate(originalProxmoxHypervisor.node.Name, p.node.Name, tempNodeName); err != nil {
		return err
	}

	if err := p.waitForTemplate(5*time.Second, 5*time.Minute, tempNodeName); err != nil {
		return err
	}

	config = map[string]interface{}{}
	config["name"] = vmImageSpec.Name

	return p.pc.ConfigureTemplate(p.node.Name, tempNodeName, config)
}

func (p *ProxmoxHypervisor) CreateImage(vms hope.VMs, vmImageSpec hope.VMImageSpec, args []string, force bool) error {
	vmDir := path.Join(vms.Root, vmImageSpec.Name)

	log.Debugf("Copying contents of %s for parameter replacement.", vmDir)
	tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, vmImageSpec.Parameters)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	allArgs := []string{"build"}
	for _, v := range args {
		allArgs = append(allArgs, "-var", v)
	}
	allArgs = append(allArgs, ".")

	packerEnvs := map[string]string{
		"PACKER_LOG":           "1",
		"PKR_VAR_proxmox_node": p.node.Name,
	}

	log.Infof("Building VM Image: %s", vmImageSpec.Name)

	return packer.ExecPackerWdEnv(tempDir, &packerEnvs, allArgs...)
}

func (p *ProxmoxHypervisor) CreateNode(node hope.Node, vms hope.VMs, vmImageSpec hope.VMImageSpec) error {
	if err := p.pc.CreateNodeFromTemplate(p.node.Name, node.Name, vmImageSpec.Name); err != nil {
		return err
	}

	if err := p.waitForNode(5*time.Second, 5*time.Minute, node.Name); err != nil {
		return err
	}

	// Have to fetch the actual network details to replace the bridge
	oConfig, err := p.pc.NodeConfiguration(p.node.Name, node.Name)
	if err != nil {
		return err
	}

	netComponents := strings.Split(oConfig.Net0, ",")
	for i, c := range netComponents {
		elems := strings.SplitN(c, "=", 2)
		if elems[0] == "bridge" {
			netComponents[i] = fmt.Sprintf("bridge=%s", node.Network)
			break
		}
	}

	config := map[string]interface{}{}
	config["cores"] = node.Cpu
	config["memory"] = node.Memory
	config["net0"] = strings.Join(netComponents, ",")

	return p.pc.ConfigureNode(p.node.Name, node.Name, config)
}

func (p *ProxmoxHypervisor) StartVM(vmName string) error {
	return p.pc.PowerOnVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) StopVM(vmName string) error {
	return p.pc.PowerOffVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) DeleteVM(vmName string) error {
	return p.pc.DeleteVmNamed(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) VMIPAddress(vmName string) (string, error) {
	return p.pc.GetNodeIP(p.node.Name, vmName)
}

func (p *ProxmoxHypervisor) waitForNode(pollInterval, timeout time.Duration, nodeName string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	deadline := time.Now().Add(timeout)

	log.Infof("Waiting for vm %s to appear on node %s", nodeName, p.node.Name)
	for {
		select {
		case <-ticker.C:
			currentVms, err := p.ListNodes()
			if err != nil {
				return err
			}

			if slices.Contains(currentVms, nodeName) {
				return nil
			}

			log.Debugf("Node %s not found yet. Waiting %s...", nodeName, pollInterval.String())
			log.Tracef("Found: %s", strings.Join(currentVms, ","))
			log.Tracef("Polling continues for %s...", time.Until(deadline).Round(time.Second).String())
		case <-timer.C:
			return fmt.Errorf("waited %s, and node %s is not yet ready", timeout.String(), nodeName)
		}
	}
}

func (p *ProxmoxHypervisor) waitForTemplate(pollInterval, timeout time.Duration, templateName string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	deadline := time.Now().Add(timeout)

	log.Infof("Waiting for template %s to appear on node %s", templateName, p.node.Name)
	for {
		select {
		case <-ticker.C:
			currentTemplates, err := p.pc.GetTemplateNames(p.node.Name)
			if err != nil {
				return err
			}

			if slices.Contains(currentTemplates, templateName) {
				return nil
			}

			log.Debugf("Template %s not found yet. Waiting %s...", templateName, pollInterval.String())
			log.Tracef("Found: %s", strings.Join(currentTemplates, ","))
			log.Tracef("Polling continues for %s...", time.Until(deadline).Round(time.Second).String())
		case <-timer.C:
			return fmt.Errorf("waited %s, and template %s is not yet ready", timeout.String(), templateName)
		}
	}
}
