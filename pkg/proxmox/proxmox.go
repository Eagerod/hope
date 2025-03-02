package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
)

type qemuApiResponse struct {
	Status   string `json:"status"`
	VmId     int    `json:"vmid"`
	Name     string `json:"name"`
	Template int    `json:"template"`
}

type ipAddressResponse struct {
	IP string `json:"ip-address"`
}

type getNetworkInterfacesResponse struct {
	Name        string              `json:"name"`
	IPAddresses []ipAddressResponse `json:"ip-addresses"`
}

// Only value that needs to be consumed right now.
type NodeConfiguration struct {
	Net0 string `json:"net0"`
}

// https://pve.proxmox.com/pve-docs/api-viewer/#
type ApiClient struct {
	User   string
	Host   string
	Token  string
	Client *http.Client
}

func NewApiClient(user, host string) *ApiClient {
	return &ApiClient{
		User:  user,
		Host:  host,
		Token: os.Getenv("PROXMOX_API_TOKEN"),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (p *ApiClient) listVMs(node string, templates bool) ([]qemuApiResponse, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu", node)
	data, err := p.request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []qemuApiResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	retVal := []qemuApiResponse{}
	for _, r := range response.Data {
		if (r.Template == 1) == templates {
			retVal = append(retVal, r)
		}
	}

	return retVal, nil
}

func (p *ApiClient) getVm(node, vmName string) (qemuApiResponse, error) {
	response, err := p.listVMs(node, false)
	if err != nil {
		return qemuApiResponse{}, err
	}

	for _, n := range response {
		if n.Name == vmName {
			return n, nil
		}
	}

	return qemuApiResponse{}, fmt.Errorf("failed to find a node named: %s", vmName)
}

func (p *ApiClient) getTemplate(node, templateName string) (qemuApiResponse, error) {
	response, err := p.listVMs(node, true)
	if err != nil {
		return qemuApiResponse{}, err
	}

	for _, n := range response {
		if n.Name == templateName {
			return n, nil
		}
	}

	return qemuApiResponse{}, fmt.Errorf("failed to find a template named: %s", templateName)
}

func (p *ApiClient) getClusterNextId() (string, error) {
	data, err := p.request("GET", "cluster/nextid", nil)
	if err != nil {
		return "", err
	}

	var response struct {
		Data string `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return "", err
	}

	return response.Data, nil
}

func (p *ApiClient) GetVmNames(node string) ([]string, error) {
	response, err := p.listVMs(node, false)
	if err != nil {
		return nil, err
	}

	retVal := []string{}
	for _, n := range response {
		retVal = append(retVal, n.Name)
	}

	return retVal, nil
}

func (p *ApiClient) GetTemplateNames(node string) ([]string, error) {
	response, err := p.listVMs(node, true)
	if err != nil {
		return nil, err
	}

	retVal := []string{}
	for _, n := range response {
		retVal = append(retVal, n.Name)
	}

	return retVal, nil
}

func (p *ApiClient) CreateNodeFromTemplate(node, vmName, templateName string) error {
	vm, err := p.getTemplate(node, templateName)
	if err != nil {
		return err
	}

	nextId, err := p.getClusterNextId()
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vm.VmId)
	params := map[string]interface{}{}
	params["full"] = true
	params["name"] = vmName
	params["newid"] = nextId
	_, err = p.request("POST", endpoint, params)
	return err
}

func (p *ApiClient) CreateNodeFromOthersTemplate(node, sourceNode, templateName string) error {
	// Process technical depends on shared vs. non-shared storage.
	// Assume unshared, and clone, then migrate.
	vm, err := p.getVm(sourceNode, templateName)
	if err != nil {
		return err
	}

	nextId, err := p.getClusterNextId()
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vm.VmId)
	params := map[string]interface{}{}
	params["full"] = true
	params["name"] = templateName
	params["newid"] = nextId
	_, err = p.request("POST", endpoint, params)
	return err
}

func (p *ApiClient) NodeConfiguration(node, vmName string) (*NodeConfiguration, error) {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/config", node, vm.VmId)
	data, err := p.request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data NodeConfiguration `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (p *ApiClient) ConfigureNode(node, vmName string, params map[string]interface{}) error {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/config", node, vm.VmId)
	_, err = p.request("PUT", endpoint, params)
	return err
}

func (p *ApiClient) GetNodeIP(node, vmName string) (string, error) {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/agent/network-get-interfaces", node, vm.VmId)
	data, err := p.request("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	var response struct {
		Data struct {
			Result []getNetworkInterfacesResponse `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return "", err
	}

	candidateIps := []string{}
	for _, n := range response.Data.Result {
		if n.Name == "lo" {
			continue
		}
		if strings.HasPrefix(n.Name, "cali") {
			continue
		}
		if strings.HasPrefix(n.Name, "tun") {
			continue
		}
		candidateIps = append(candidateIps, n.IPAddresses[0].IP)
	}

	if len(candidateIps) == 0 {
		return "", fmt.Errorf("failed to find a non-loopback, non-tunnel, non-kubernetes IP for %s", vmName)
	}
	if len(candidateIps) == 1 {
		return candidateIps[0], nil
	}

	return "", fmt.Errorf("found multiple possible IPs for %s", vmName)
}

func (p *ApiClient) PowerOnVmNamed(node, vmName string) error {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "running" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/start", node, vm.VmId)
	_, err = p.request("POST", endpoint, nil)
	return err
}

func (p *ApiClient) PowerOffVmNamed(node, vmName string) error {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "stopped" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/stop", node, vm.VmId)
	_, err = p.request("POST", endpoint, nil)
	return err
}

func (p *ApiClient) DeleteVmNamed(node, vmName string) error {
	vm, err := p.getVm(node, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d", node, vm.VmId)
	_, err = p.request("DELETE", endpoint, nil)
	return err
}

func (p *ApiClient) request(method, endpoint string, params interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", p.Host, endpoint)
	log.Tracef("%s: %s", method, url)

	var body io.Reader = nil
	hasBody := false
	if params != nil {
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(jsonBytes)
		hasBody = true
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	pveToken := fmt.Sprintf("PVEAPIToken=%s!%s", p.User, p.Token)
	req.Header.Set("Authorization", pveToken)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response error from %s, %d: %s", req.Host, resp.StatusCode, data)
	}

	return data, nil
}
