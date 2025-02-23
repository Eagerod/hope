package proxmox

import (
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
	data, err := proxmoxRequest(user, host, endpoint, nil)
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

func GetNodeIP(user, node, host, vmName string) (string, error) {
	vm, err := getVm(user, node, host, vmName)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/agent/network-get-interfaces", node, vm.VmId)
	data, err := proxmoxRequest(user, host, endpoint, nil)
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

func proxmoxRequest(user, host, endpoint string, params map[string]string) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/%s", host, endpoint)
	log.Trace(url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("response error from %s, %d: %s", host, resp.StatusCode, body)
	}

	return body, nil
}
