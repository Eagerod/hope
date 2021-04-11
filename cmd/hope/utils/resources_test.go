package utils

import (
	"testing"
)

import (
	"github.com/Eagerod/hope/pkg/hope"
	"github.com/stretchr/testify/assert"
)

var testResources []hope.Resource = []hope.Resource{
	{
		Name: "calico",
		File: "https://docs.projectcalico.org/manifests/calico.yaml",
		Tags: []string{"network"},
	},
	{
		Name: "load-balancer-namespace",
		File: "https://raw.githubusercontent.com/metallb/metallb/v0.9.5/manifests/namespace.yaml",
		Tags: []string{"network", "load-balancer"},
	},
	{
		Name:       "load-balancer-config",
		Inline:     "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  namespace: metallb-system\n  name: config\ndata:\n  config: |\n    address-pools:\n    - name: default\n      protocol: layer2\n      addresses:\n      - 192.168.1.16-192.168.1.24\n---\napiVersion: v1\ndata:\n  secretkey: ${METALLB_SYSTEM_MEMBERLIST_SECRET_KEY}\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  name: memberlist\n  namespace: metallb-system\n",
		Parameters: []string{"METALLB_SYSTEM_MEMBERLIST_SECRET_KEY"},
		Tags:       []string{"network"},
	},
	{
		Name: "build-some-image",
		Build: hope.BuildSpec{
			Path: "some-dir-with-dockerfile",
			Pull: "always",
			Tag:  "registry.internal.aleemhaji.com/example-repo:latest",
		},
		Tags: []string{"app1"},
	},
	{
		Name: "copy-some-image",
		Build: hope.BuildSpec{
			Source: "python:3.7",
			Pull:   "if-not-present",
			Tag:    "registry.internal.aleemhaji.com/python:3.7",
		},
		Tags: []string{"dockercache"},
	},
	{
		Name: "database",
		File: "test/mysql.yaml",
		Tags: []string{"database"},
	},
	{
		Name: "wait-for-some-kind-of-job",
		Job:  "init-the-database",
		Tags: []string{"database"},
	},
	{
		Name: "exec-in-a-running-pod",
		Exec: hope.ExecSpec{
			Selector: "deploy/mysql",
			Timeout:  "60s",
			Command:  []string{"mysql", "--database", "test", "-e", "select * from abc;"},
		},
		Tags: []string{"database"},
	},
}

// Basically a smoke test, don't want to define a ton of yaml blocks to test
//   this extensively quite yet.
func TestGetResources(t *testing.T) {
	resetViper(t)

	resources, err := GetResources()
	assert.Nil(t, err)
	assert.Equal(t, testResources, *resources)
}

func TestGetIdentifiableResources(t *testing.T) {
	resetViper(t)

	multipleNamesResult := []hope.Resource{}
	multipleNamesResult = append(multipleNamesResult, testResources[0], testResources[2])

	multipleTagsResult := []hope.Resource{}
	multipleTagsResult = append(multipleTagsResult, testResources[0:3]...)
	multipleTagsResult = append(multipleTagsResult, testResources[5:8]...)

	tagAndNameResult := []hope.Resource{}
	tagAndNameResult = append(tagAndNameResult, testResources[0])
	tagAndNameResult = append(tagAndNameResult, testResources[5:8]...)

	var tests = []struct {
		name     string
		names    []string
		tags     []string
		expected []hope.Resource
	}{
		{"No matches", []string{}, []string{}, []hope.Resource{}},
		{"Only name", []string{"calico"}, []string{}, testResources[0:1]},
		{"Multiple names", []string{"calico", "load-balancer-config"}, []string{}, multipleNamesResult},
		{"Only tag", []string{}, []string{"network"}, testResources[0:3]},
		{"Multiple tags", []string{}, []string{"network", "database"}, multipleTagsResult},
		{"Tag and name", []string{"calico"}, []string{"database"}, tagAndNameResult},
		{"Tag and name overlap", []string{"calico"}, []string{"network"}, testResources[0:3]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := GetIdentifiableResources(&tt.names, &tt.tags)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, *resources)
		})
	}
}
