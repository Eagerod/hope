package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

import (
	log "github.com/sirupsen/logrus"
)

func GetNodes(user, node, host string) ([]string, error) {
	data, err := proxmoxRequest(user, node, host, "qemu", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	retVal := []string{}
	for _, n := range response.Data {
		retVal = append(retVal, n.Name)
	}

	return retVal, nil
}

func proxmoxRequest(user, node, host, endpoint string, params map[string]string) ([]byte, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/%s", host, node, endpoint)
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
