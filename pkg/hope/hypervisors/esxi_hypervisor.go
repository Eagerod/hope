package hypervisors

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
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

	insecure bool
}

func (hyp *EsxiHypervisor) Initialize(node hope.Node) error {
	hyp.node = node

	errs := []error{}
	for _, s := range node.Parameters {
		key, value, _ := strings.Cut(s, "=")
		switch key {
		case "INSECURE":
			switch value {
			case "true", "1":
				hyp.insecure = true
			case "false", "0":
				hyp.insecure = false
			default:
				errs = append(errs, fmt.Errorf("unknown value '%s' for INSECURE in ESXI hypervisor", value))
			}
		default:
			errs = append(errs, fmt.Errorf("unknown property '%s' in ESXI hypervisor", key))
		}
	}

	return errors.Join(errs...)
}

func (hyp *EsxiHypervisor) CopyImageMode() CopyImageMode {
	return CopyImageModeToAll
}

func (hyp *EsxiHypervisor) ListNodes() ([]string, error) {
	v, e := esxi.ListVms(hyp.node.ConnectionString())
	if e == nil {
		return *v, nil
	}
	return nil, e
}

func (hyp *EsxiHypervisor) ListBuiltImages(vms hope.VMs) ([]string, error) {
	imageDirectories := []string{}

	entries, err := os.ReadDir(vms.Output)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		hasDisk := false
		hasVM := false

		files, err := os.ReadDir(path.Join(vms.Output, e.Name()))
		if err != nil {
			return nil, err
		}

		for _, fn := range files {
			ext := filepath.Ext(fn.Name())
			if ext == ".vmdk" {
				hasDisk = true
			} else if ext == ".ovf" || ext == ".vmx" {
				hasVM = true
			}
		}

		if hasDisk && hasVM {
			imageDirectories = append(imageDirectories, e.Name())
		}
	}

	return imageDirectories, nil
}

func (hyp *EsxiHypervisor) ListAvailableImages(vms hope.VMs) ([]string, error) {
	remoteVmfsPath := path.Join("/", "vmfs", "volumes", hyp.node.Datastore, "ovfs")

	output, err := ssh.GetSSH(hyp.node.ConnectionString(), "find", remoteVmfsPath, "-type", "d", "-maxdepth", "1")
	if err != nil {
		return nil, err
	}

	retVal := []string{}
	trimPrefix := remoteVmfsPath + "/"
	for _, fullpath := range strings.Split(output, "\n") {
		relPath := strings.TrimPrefix(fullpath, trimPrefix)
		retVal = append(retVal, relPath)
	}

	return retVal, nil
}

func (hyp *EsxiHypervisor) ResolveNode(node hope.Node) (hope.Node, error) {
	ip, err := esxi.GetIpAddressOfVmNamed(hyp.node.ConnectionString(), node.Name)
	if err != nil {
		return hope.Node{}, err
	}

	ip = strings.TrimSpace(ip)
	if ip == "0.0.0.0" {
		return hope.Node{}, fmt.Errorf("failed to find IP for vm %s on %s", node.Name, hyp.node.Name)
	}

	node.Hypervisor = ""
	node.Host = ip
	return node, nil
}

func (hyp *EsxiHypervisor) UnderlyingNode() (hope.Node, error) {
	return hyp.node, nil
}

func (hyp *EsxiHypervisor) CreateNode(node hope.Node, vms hope.VMs, vmImageSpec hope.VMImageSpec) error {
	packerSpec, err := hyp.renderedPackerSpec(vms, vmImageSpec)
	if err != nil {
		return err
	}

	// Exec OVF tool to create VM.
	// https://www.virtuallyghetto.com/2012/05/how-to-deploy-ovfova-in-esxi-shell.html
	sourceNetworkName, ok := packerSpec.Builders[0].VMXData["ethernet0.networkName"]
	if !ok {
		return fmt.Errorf("failed to find network definition in VM spec: %s", node.Name)
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
	}

	if hyp.insecure {
		allArgs = append(allArgs, "--noSSLVerify")
	}

	allArgs = append(allArgs, []string{
		remoteOvfPath,
		"vi://root@localhost",
	}...)

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

func (hyp *EsxiHypervisor) CopyImage(vms hope.VMs, vmImageSpec hope.VMImageSpec, srcHypervisor Hypervisor) error {
	packerSpec, err := hyp.renderedPackerSpec(vms, vmImageSpec)
	if err != nil {
		return err
	}

	outputDirectory := ""
	for _, builder := range packerSpec.Builders {
		if builder.Type == "vmware-iso" {
			if outputDirectory != "" {
				return fmt.Errorf("spec %s has multiple vmware-iso output directories", vmImageSpec.Name)
			}
			outputDirectory = builder.OutputDirectory
		}
	}

	connectionString := hyp.node.ConnectionString()
	remoteVmfsPath := path.Join("/", "vmfs", "volumes", hyp.node.Datastore, "ovfs", vmImageSpec.Name)
	remoteVMPath := fmt.Sprintf("%s:%s", connectionString, remoteVmfsPath)

	if err := ssh.ExecSSH(connectionString, "rm", "-rf", remoteVmfsPath); err != nil {
		return err
	}

	if err := scp.ExecSCP("-pr", outputDirectory, remoteVMPath); err != nil {
		return err
	}

	return nil
}

func (hyp *EsxiHypervisor) CreateImage(vms hope.VMs, vmImageSpec hope.VMImageSpec, args []string, force bool) error {
	vmDir := path.Join(vms.Root, vmImageSpec.Name)
	outputDir := path.Join(vms.Output, vmImageSpec.Name)
	log.Tracef("Looking for VM definition in %s", vmDir)

	// This is done in advance so that the error can show the user the
	//   real path the file that's expected to load, rather than a path in
	//   the temp directory everything gets copied into.
	packerJsonPath := path.Join(vmDir, "packer.json")
	if _, err := os.Stat(packerJsonPath); err != nil && os.IsNotExist(err) {
		return fmt.Errorf("VM packer file not found at path: %s", packerJsonPath)
	} else if err != nil {
		return err
	}

	// Create full parameter set.
	allParameters := append(vmImageSpec.Parameters,
		fmt.Sprintf("ESXI_HOST=%s", hyp.node.Host),
		fmt.Sprintf("ESXI_USERNAME=%s", hyp.node.User),
		fmt.Sprintf("ESXI_DATASTORE=%s", hyp.node.Datastore),
		fmt.Sprintf("OUTPUT_DIR=%s", outputDir),
	)

	log.Debugf("Copying contents of %s for parameter replacement.", vmDir)
	tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, allParameters)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Check caches to see if I even want to build this again.
	tempPackerJsonPath := path.Join(tempDir, "packer.json")
	packerSpec, err := packer.SpecFromPath(tempPackerJsonPath)
	if err != nil {
		return err
	}

	// Packer runs out of temp dir, so directories have to be absolute.
	packerOutDir := packerSpec.Builders[0].OutputDirectory
	if !path.IsAbs(packerOutDir) {
		return fmt.Errorf("packer output directory %s must be absolute", packerOutDir)
	}

	if !path.IsAbs(vms.Cache) {
		return fmt.Errorf("packer cache directory %s must be absolute", vms.Cache)
	}

	if force {
		log.Infof("Deleting %s", packerOutDir)
		os.RemoveAll(packerOutDir)
	} else {
		stat, err := os.Stat(packerOutDir)
		if err != nil && os.IsNotExist(err) {
			log.Debugf("Will create a new directory at %s...", packerOutDir)
		} else if err != nil {
			return err
		} else {
			if !stat.IsDir() {
				return fmt.Errorf("file exists at path %s", packerOutDir)
			}

			files, err := os.ReadDir(packerOutDir)
			if err != nil {
				return err
			}

			if len(files) != 0 {
				return fmt.Errorf("directory at path %s already exists and is not empty", packerOutDir)
			}
		}
	}

	// Try to create a file in the same directory as the output will be.
	// Prevents going through the whole process when the output directory
	//   isn't writable.
	// Seems like a no brainer for packer to do that check.
	if err := os.MkdirAll(packerOutDir, 0755); err != nil {
		return fmt.Errorf("directory at path %s is not writable; %w", packerOutDir, err)
	}

	allArgs := []string{"build"}
	for _, v := range args {
		allArgs = append(allArgs, "-var", v)
	}
	allArgs = append(allArgs, tempPackerJsonPath)

	packerEsxiVncProbeTimeout := os.Getenv("PACKER_ESXI_VNC_PROBE_TIMEOUT")
	if packerEsxiVncProbeTimeout == "" {
		log.Info("PACKER_ESXI_VNC_PROBE_TIMEOUT not set, defaulting to 2s")
		packerEsxiVncProbeTimeout = "2s"
	}

	packerEnvs := map[string]string{
		"PACKER_CACHE_DIR":              vms.Cache,
		"PACKER_LOG":                    "1",
		"PACKER_ESXI_VNC_PROBE_TIMEOUT": packerEsxiVncProbeTimeout,
	}

	log.Infof("Building VM Image: %s", vmImageSpec.Name)
	return packer.ExecPackerWdEnv(tempDir, &packerEnvs, allArgs...)
}

func (hyp *EsxiHypervisor) DeleteVM(name string) error {
	// If the VM is on, don't allow the user to proceed, and force them to
	//   shut it off themselves.
	connectionString := hyp.node.ConnectionString()
	powerState, err := esxi.PowerStateOfVmNamed(connectionString, name)
	if err != nil {
		return err
	}

	if powerState != esxi.VmStatePoweredOff {
		return fmt.Errorf("VM %s has power state: %s; cannot delete", name, powerState)
	}

	return esxi.DeleteVmNamed(connectionString, name)
}

func (hyp *EsxiHypervisor) VMIPAddress(name string) (string, error) {
	ip, err := esxi.GetIpAddressOfVmNamed(hyp.node.ConnectionString(), name)
	if err != nil {
		return "", err
	}

	ip = strings.TrimSpace(ip)
	if ip == "0.0.0.0" {
		return "", fmt.Errorf("VM %s hasn't bound an IP address yet", name)
	}

	return ip, nil
}

func (hyp *EsxiHypervisor) StartVM(name string) error {
	return esxi.PowerOnVmNamed(hyp.node.ConnectionString(), name)

}

func (hyp *EsxiHypervisor) StopVM(name string) error {
	return esxi.PowerOffVmNamed(hyp.node.ConnectionString(), name)
}

func (hyp *EsxiHypervisor) renderedPackerSpec(vms hope.VMs, vmImageSpec hope.VMImageSpec) (*packer.JsonSpec, error) {
	vmDir := path.Join(vms.Root, vmImageSpec.Name)
	outputDir := path.Join(vms.Output, vmImageSpec.Name)

	// Create full parameter set.
	allParameters := append(vmImageSpec.Parameters,
		fmt.Sprintf("ESXI_HOST=%s", hyp.node.Host),
		fmt.Sprintf("ESXI_USERNAME=%s", hyp.node.User),
		fmt.Sprintf("ESXI_DATASTORE=%s", hyp.node.Datastore),
		fmt.Sprintf("OUTPUT_DIR=%s", outputDir),
	)

	log.Debugf("Copying contents of %s for parameter replacement.", vmDir)
	tempDir, err := hope.ReplaceParametersInDirectoryCopy(vmDir, allParameters)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	tempPackerJsonPath := path.Join(tempDir, "packer.json")
	return packer.SpecFromPath(tempPackerJsonPath)
}
