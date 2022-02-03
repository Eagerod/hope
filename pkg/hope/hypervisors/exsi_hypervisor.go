package hypervisors

import (
	"fmt"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Eagerod/hope/pkg/esxi"
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/Eagerod/hope/pkg/packer"
	"github.com/Eagerod/hope/pkg/scp"
	"github.com/Eagerod/hope/pkg/ssh"
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

func (hyp *EsxiHypervisor) CreateNode(node hope.Node, vms hope.VMs, vmImageSpec hope.VMImageSpec) error {
	vmDir := path.Join(vms.Root, vmImageSpec.Name)

	log.Debug(fmt.Sprintf("Copying contents of %s for parameter replacement.", vmDir))
	tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, vmImageSpec.Parameters)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	tempPackerJsonPath := path.Join(tempDir, "packer.json")
	packerSpec, err := packer.SpecFromPath(tempPackerJsonPath)
	if err != nil {
		return err
	}

	// Exec OVF tool to create VM.
	// https://www.virtuallyghetto.com/2012/05/how-to-deploy-ovfova-in-esxi-shell.html
	sourceNetworkName, ok := packerSpec.Builders[0].VMXData["ethernet0.networkName"]
	if !ok {
		return fmt.Errorf("Failed to find network definition in VM spec: %s", node.Name)
	}

	datastoreRoot := path.Join("/", "vmfs", "volumes", hyp.node.Datastore)
	vmOvfName := fmt.Sprintf("%s.ovf", packerSpec.Builders[0].VMName)
	remoteOvfPath := path.Join(datastoreRoot, "ovfs", packerSpec.Builders[0].VMName, vmOvfName)
	allArgs := []string{
		hyp.node.ConnectionString(),
		path.Join(datastoreRoot, "bin", "ovftool", "ovftool"),
		"--diskMode=thin",
		fmt.Sprintf("--datastore=%s", hyp.node.Datastore),
		fmt.Sprintf("--name=%s", node.Name),
		fmt.Sprintf("--net:'%s=%s'", sourceNetworkName, hyp.node.Network),
		fmt.Sprintf("--numberOfCpus:'*'=%d", node.Cpu),
		fmt.Sprintf("--memorySize:'*'=%d", node.Memory),
		remoteOvfPath,
		"vi://root@localhost",
	}

	// Check to see if the ESXI_ROOT_PASSWORD environment if set.
	// If so, pass it on to the ssh invocation to help limit user
	//   interaction.
	esxiRootPassword := os.Getenv("ESXI_ROOT_PASSWORD")
	if esxiRootPassword == "" {
		log.Warn("ESXI_ROOT_PASSWORD not provided. A password prompt will need to be filled.")
		return ssh.ExecSSH(allArgs...)
	} else {
		stdin := fmt.Sprintf("%s\n", esxiRootPassword)
		return ssh.ExecSSHStdin(stdin, allArgs...)
	}
}

func (hyp *EsxiHypervisor) CopyImage(packerSpec packer.JsonSpec, vm hope.VMs, vmImageSpec hope.VMImageSpec) error {
	for _, builder := range packerSpec.Builders {
		if builder.Type != "vmware-iso" {
			continue
		}

		connectionString := hyp.node.ConnectionString()
		remoteVmfsPath := path.Join("/", "vmfs", "volumes", hyp.node.Datastore, "ovfs", vmImageSpec.Name)
		remoteVMPath := fmt.Sprintf("%s:%s", connectionString, remoteVmfsPath)

		if err := ssh.ExecSSH(connectionString, "rm", "-rf", remoteVmfsPath); err != nil {
			return err
		}

		if err := scp.ExecSCP("-pr", builder.OutputDirectory, remoteVMPath); err != nil {
			return err
		}
	}

	return nil
}
