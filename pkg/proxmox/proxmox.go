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
	Status string `json:"status"`
	VmId   int    `json:"vmid"`
	Name   string `json:"name"`
}

type ipAddressResponse struct {
	IP string `json:"ip-address"`
}

type getNetworkInterfacesResponse struct {
	Name        string              `json:"name"`
	IPAddresses []ipAddressResponse `json:"ip-addresses"`
}

type ProxmoxApiClient struct {
	User   string
	Host   string
	Token  string
	Client *http.Client
}

func NewProxmoxApiClient(user, host string) *ProxmoxApiClient {
	return &ProxmoxApiClient{
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

func (p *ProxmoxApiClient) listVMs(node string) ([]qemuApiResponse, error) {
	data, err := p.request("GET", fmt.Sprintf("nodes/%s/qemu", node), nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []qemuApiResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (p *ProxmoxApiClient) getVm(node, vmName string) (qemuApiResponse, error) {
	response, err := p.listVMs(node)
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

func GetNodes(user, node, host string) ([]string, error) {
	pc := NewProxmoxApiClient(user, host)
	response, err := pc.listVMs(node)
	if err != nil {
		return nil, err
	}

	retVal := []string{}
	for _, n := range response {
		retVal = append(retVal, n.Name)
	}

	return retVal, nil
}

func CreateNodeFromTemplate(user, node, host, vmName, templateName string) error {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, templateName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vm.VmId)
	params := map[string]interface{}{}
	params["full"] = true
	params["name"] = vmName
	_, err = pc.request("POST", endpoint, params)
	return err
}

func ConfigureNode(user, node, host, vmName string, params map[string]interface{}) error {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/config", node, vm.VmId)
	_, err = pc.request("PUT", endpoint, params)
	return err
}

func GetNodeIP(user, node, host, vmName string) (string, error) {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, vmName)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/agent/network-get-interfaces", node, vm.VmId)
	data, err := pc.request("GET", endpoint, nil)
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

func PowerOnVmNamed(user, node, host, vmName string) error {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "running" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/start", node, vm.VmId)
	_, err = pc.request("POST", endpoint, nil)
	return err
}

func PowerOffVmNamed(user, node, host, vmName string) error {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "stopped" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/stop", node, vm.VmId)
	_, err = pc.request("POST", endpoint, nil)
	return err
}

func DeleteVmNamed(user, node, host, vmName string) error {
	pc := NewProxmoxApiClient(user, host)
	vm, err := pc.getVm(node, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d", node, vm.VmId)
	_, err = pc.request("DELETE", endpoint, nil)
	return err
}

func (p *ProxmoxApiClient) request(method, endpoint string, params interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", p.Host, endpoint)
	log.Trace(url)

	var body io.Reader = nil
	if params != nil {
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s!%s", p.User, p.Token))
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

func proxmoxDoRequest(user string, req *http.Request) ([]byte, error) {
	token := os.Getenv("PROXMOX_API_TOKEN")
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s!%s", user, token))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response error from %s, %d: %s", req.Host, resp.StatusCode, body)
	}

	return body, nil
}
