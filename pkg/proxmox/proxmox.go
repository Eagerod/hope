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

// TODO: Dedicated Proxmox client kind of deal?

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

func basicVmInformation(user, node, host string) ([]qemuApiResponse, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu", node)
	data, err := proxmoxGetRequest(user, host, endpoint)
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

func getVm(user, node, host, vmName string) (qemuApiResponse, error) {
	response, err := basicVmInformation(user, node, host)
	if err != nil {
		return qemuApiResponse{}, err
	}

	for _, n := range response {
		if n.Name == vmName {
			return n, nil
		}
	}

	return qemuApiResponse{}, fmt.Errorf("Failed to find a node named: %s", vmName)
}

func GetNodes(user, node, host string) ([]string, error) {
	response, err := basicVmInformation(user, node, host)
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
	vm, err := getVm(user, node, host, templateName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vm.VmId)
	params := map[string]interface{}{}
	params["full"] = true
	params["name"] = vmName
	_, err = proxmoxPostRequest(user, host, endpoint, params)
	return err
}

func ConfigureNode(user, node, host, vmName string, params map[string]interface{}) error {
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/config", node, vm.VmId)
	_, err = proxmoxPutRequest(user, host, endpoint, params)
	return err
}

func GetNodeIP(user, node, host, vmName string) (string, error) {
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/agent/network-get-interfaces", node, vm.VmId)
	data, err := proxmoxGetRequest(user, host, endpoint)
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
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "running" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/start", node, vm.VmId)
	_, err = proxmoxPostRequest(user, host, endpoint, nil)
	return err
}

func PowerOffVmNamed(user, node, host, vmName string) error {
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return err
	}

	if vm.Status == "stopped" {
		return nil
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/stop", node, vm.VmId)
	_, err = proxmoxPostRequest(user, host, endpoint, nil)
	return err
}

func DeleteVmNamed(user, node, host, vmName string) error {
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d", node, vm.VmId)
	_, err = proxmoxDeleteRequest(user, host, endpoint)
	return err
}

func proxmoxGetRequest(user, host, endpoint string) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", host, endpoint)
	log.Trace(url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return proxmoxDoRequest(user, req)
}

func proxmoxDeleteRequest(user, host, endpoint string) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", host, endpoint)
	log.Trace(url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	return proxmoxDoRequest(user, req)
}

func proxmoxPostRequest(user, host, endpoint string, params interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", host, endpoint)
	log.Trace(url)

	var body io.Reader = nil
	if params != nil {
		pureBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(pureBytes)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	return proxmoxDoRequest(user, req)
}

func proxmoxPutRequest(user, host, endpoint string, params interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", host, endpoint)
	log.Trace(url)

	var body io.Reader = nil
	if params != nil {
		pureBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(pureBytes)
	}

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	return proxmoxDoRequest(user, req)
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
