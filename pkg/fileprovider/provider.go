package fileprovider

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
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

	clustersLock  sync.RWMutex
	clusters      map[string]cluster.Cluster
	clusterCancel map[string]func()
	// Using a map of errors instead of a channel to be able to collect
	// the errors in RunOnce without blocking.
	clusterErrs map[string]error
}

func newProvider() *Provider {
	return &Provider{
		UpdateInterval: 1 * time.Second,
		clusters:       make(map[string]cluster.Cluster),
		clusterCancel:  make(map[string]func()),
		clusterErrs:    make(map[string]error),
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
	if err := p.run(ctx); err != nil {
		return fmt.Errorf("initial update failed: %w", err)
	}
	for range time.Tick(p.UpdateInterval) {
		if ctx.Err() != nil {
			break
		}
		if err := p.run(ctx); err != nil {
			return fmt.Errorf("failed to update clusters: %w", err)
		}
	}
	return p.collectErrors()
}

func (p *Provider) RunOnce(ctx context.Context) error {
	if err := p.run(ctx); err != nil {
		return err
	}
	return p.collectErrors()
}

func (p *Provider) run(ctx context.Context) error {
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
	defer p.clustersLock.Unlock()
	// add new clusters
	for name, cl := range c {
		if _, ok := p.clusters[name]; ok {
			continue
		}
		ctx, cancel := context.WithCancel(ctx)
		p.clusters[name] = cl
		p.clusterCancel[name] = cancel
		go func() {
			if err := cl.Start(ctx); err != nil {
				if ctx.Err() == nil || ctx.Err() == context.Canceled {
					return
				}

				// If .Start returns an error remove the cluster from
				// the provider and store teh error in the clusterErrs
				// map. The provider will pick the error up.
				p.clustersLock.Lock()
				delete(p.clusters, name)
				delete(p.clusterCancel, name)
				p.clusterErrs[name] = errors.Join(p.clusterErrs[name], err)
				p.clustersLock.Unlock()
			}
		}()
	}

	// delete clusters that are no longer present
	for name := range p.clusters {
		if _, ok := c[name]; ok {
			continue
		}
		cancel := p.clusterCancel[name]
		cancel()
		delete(p.clusters, name)
		delete(p.clusterCancel, name)
		// keep clusterErrs
	}
	return nil
}

func (p *Provider) collectErrors() error {
	p.clustersLock.Lock()
	defer p.clustersLock.Unlock()

	var err error
	for name, clusterErr := range p.clusterErrs {
		if clusterErr == nil {
			continue
		}
		err = errors.Join(err, fmt.Errorf("cluster %q client errored: %w", name, clusterErr))
		delete(p.clusterErrs, name)
	}
	return err
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

// ClusterNames returns the names of all clusters known to the provider.
func (p *Provider) ClusterNames() []string {
	p.clustersLock.RLock()
	defer p.clustersLock.RUnlock()
	return slices.Sorted(maps.Keys(p.clusters))
}
