package mkl

import (
	"io"
	"os"

	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Config struct {
	StatusStyle map[ResourceStatus]string `json:"statusStyle,omitempty"`
	Nodes       map[string]Node           `json:"nodes,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		StatusStyle: map[ResourceStatus]string{
			Absent:  "stroke:#808080",
			Pending: "stroke:#FFFF00",
			Healthy: "stroke:#00FF00",
		},
		Nodes: make(map[string]Node),
	}
}

func ParseFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Config, error) {
	config := DefaultConfig()
	if err := yaml.UnmarshalStrict(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

type Nodes map[string]Node

type Node struct {
	Selector           NodeSelector `json:"selector,omitempty"`
	HealthyWhenPresent bool         `json:"healthyWhenPresent,omitempty"`
	HealthType         string       `json:"healthType,omitempty"`
}

type NodeSelector struct {
	Cluster       string                      `json:"cluster,omitempty"`
	Namespace     string                      `json:"namespace,omitempty"`
	GVR           schema.GroupVersionResource `json:"gvr,omitempty"`
	Name          string                      `json:"name,omitempty"`
	LabelSelector metav1.LabelSelector        `json:"labelSelector,omitempty"`
	Owner         OwnerReference              `json:"owner,omitempty"`
}

type OwnerReference struct {
	GVR  schema.GroupVersionResource `json:"gvr,omitempty"`
	Name string                      `json:"name,omitempty"`
}
