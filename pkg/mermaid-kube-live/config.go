package mermaidkubelive

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Config struct {
	Nodes map[string]Node `json:"nodes,omitempty"`
}

func ParseFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
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
	LabelSelector metav1.LabelSelector        `json:"labels,omitempty"`
}
