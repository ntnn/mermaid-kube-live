package v1alpha1

import (
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

// ParseFile parses the given YAML file.
func ParseFile(filename string) (*Config, error) {
	f, err := os.Open(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	config, err := Parse(data)
	if err != nil {
		return nil, err
	}

	return config, f.Close()
}

// Parse parses the given YAML data.
func Parse(data []byte) (*Config, error) {
	config := &Config{}
	if err := yaml.UnmarshalStrict(data, config); err != nil {
		return nil, err
	}

	return config, nil
}
