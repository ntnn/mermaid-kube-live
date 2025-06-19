package fileprovider

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

var _ multicluster.Provider = &Provider{}

// Provider implements multicluster.Provider using kubeconfig files.
type Provider struct {
	clusters map[string]cluster.Cluster
}

func FromContexts(kubeCtxs map[string]*rest.Config) (*Provider, error) {
	provider := &Provider{
		clusters: make(map[string]cluster.Cluster, len(kubeCtxs)),
	}

	for name, kubeCtx := range kubeCtxs {
		cl, err := cluster.New(kubeCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster for context %q: %w", name, err)
		}
		provider.clusters[name] = cl
	}

	return provider, nil
}

func FromFiles(filepaths ...string) (*Provider, error) {
	kubeCtxs := map[string]*rest.Config{}

	for _, filepath := range filepaths {
		fileKubeCtxs, err := ReadContextsFromFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to read kubeconfig from file %q: %w", filepath, err)
		}
		for name, kubeCtx := range fileKubeCtxs {
			if _, exists := kubeCtxs[name]; exists {
				return nil, fmt.Errorf("duplicate context name %q found in file %q", name, filepath)
			}
			kubeCtxs[name] = kubeCtx
		}
	}

	return FromContexts(kubeCtxs)
}

var KubeconfigGlobs = []string{"*.kubeconfig", "*.kubeconfig.yaml", "*.kubeconfig.yml"}

func FromDirectory(dirpath string) (*Provider, error) {
	matches := []string{}

	for _, glob := range KubeconfigGlobs {
		globMatches, err := filepath.Glob(filepath.Join(dirpath, glob))
		if err != nil {
			return nil, fmt.Errorf("failed to glob files in directory %q with pattern %q: %w", dirpath, glob, err)
		}
		matches = append(matches, globMatches...)
	}

	return FromFiles(matches...)
}

func (p *Provider) Run(ctx context.Context, _ mcmanager.Manager) error {
	<-ctx.Done()
	return nil
}

// Get returns the cluster with the given name.
// If the cluster name is empty (""), it returns the first cluster
// found.
func (p *Provider) Get(_ context.Context, clusterName string) (cluster.Cluster, error) {
	if clusterName == "" {
		for _, cl := range p.clusters {
			return cl, nil
		}
	}

	cl, ok := p.clusters[clusterName]
	if !ok {
		return nil, multicluster.ErrClusterNotFound
	}
	return cl, nil
}

func (p *Provider) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	for name, cl := range p.clusters {
		if err := cl.GetCache().IndexField(ctx, obj, field, extractValue); err != nil {
			return fmt.Errorf("failed to index field %q on cluster %q: %w", field, name, err)
		}
	}
	return nil
}
