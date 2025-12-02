package v1alpha1

import (
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

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
	config := &Config{}
	if err := yaml.UnmarshalStrict(data, config); err != nil {
		return nil, err
	}
	return config, nil
}
