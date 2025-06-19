package mermaidkubelive

import (
	"context"
	"fmt"

	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

type MKL struct {
	config   *Config
	provider multicluster.Provider
}

func NewMKL(config *Config, provider multicluster.Provider) *MKL {
	return &MKL{
		config:   config,
		provider: provider,
	}
}

func (m *MKL) SearchResources(ctx context.Context) (map[string]ResourceState, error) {
	ret := make(map[string]ResourceState, len(m.config.Nodes))

	for name, node := range m.config.Nodes {
		cluster, err := m.provider.Get(ctx, node.Selector.Cluster)
		if err != nil {
			return nil, err
		}

		state, err := GetResourceState(ctx, cluster.GetConfig(), node)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource state for node %s: %w", name, err)
		}

		ret[name] = state
	}

	return ret, nil
}
