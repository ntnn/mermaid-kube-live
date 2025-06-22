package fileprovider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

var _ multicluster.Provider = &Provider{}

// Provider implements multicluster.Provider using kubeconfig files.
type Provider struct {
	UpdateInterval time.Duration

	directory string
	filePaths []string

	clustersLock sync.RWMutex
	clusters     map[string]cluster.Cluster
}

func newProvider() *Provider {
	return &Provider{
		UpdateInterval: 1 * time.Second,
		clusters:       make(map[string]cluster.Cluster),
	}
}

func FromDirectory(dirpath string) (*Provider, error) {
	p := newProvider()
	p.directory = dirpath
	return p, nil
}

func FromFiles(filepaths ...string) (*Provider, error) {
	p := newProvider()
	p.filePaths = filepaths
	return p, nil
}

func (p *Provider) Run(ctx context.Context) error {
	if err := p.RunOnce(ctx); err != nil {
		return fmt.Errorf("initial update failed: %w", err)
	}
	for range time.Tick(p.UpdateInterval) {
		if ctx.Err() != nil {
			return nil
		}
		if err := p.RunOnce(ctx); err != nil {
			return fmt.Errorf("failed to update clusters: %w", err)
		}
	}
	return nil
}

func (p *Provider) RunOnce(ctx context.Context) error {
	var c clusters
	var err error
	if p.directory != "" {
		c, err = fromDirectory(p.directory)
	} else {
		c, err = fromFiles(p.filePaths...)
	}
	if err != nil {
		return fmt.Errorf("failed to load clusters: %w", err)
	}

	p.clustersLock.Lock()
	p.clusters = c
	p.clustersLock.Unlock()
	return nil
}

// Get returns the cluster with the given name.
// If the cluster name is empty (""), it returns the first cluster
// found.
func (p *Provider) Get(_ context.Context, clusterName string) (cluster.Cluster, error) {
	p.clustersLock.RLock()
	defer p.clustersLock.RUnlock()

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
	p.clustersLock.RLock()
	defer p.clustersLock.RUnlock()

	for name, cl := range p.clusters {
		if err := cl.GetCache().IndexField(ctx, obj, field, extractValue); err != nil {
			return fmt.Errorf("failed to index field %q on cluster %q: %w", field, name, err)
		}
	}
	return nil
}
