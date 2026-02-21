// Package multiplexer contains a wrapper that multiplexes cluster.Cluster it
// receives from e.g. a multicluster.Provider to multiple
// multicluster.Aware.
package multiplexer

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	mctrl "sigs.k8s.io/multicluster-runtime"
	"sigs.k8s.io/multicluster-runtime/pkg/clusters"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

// Multiplexer is a wrapper that multiplexes cluster.Cluster it receives
// from e.g. a multicluster.Provider to multiple multicluster.Aware. It
// also forwards already known cluster.Cluster to new multicluster.Aware.
type Multiplexer struct {
	Logger   logr.Logger
	lock     sync.Mutex
	Registry *clusters.Registry[cluster.Cluster]
	awares   map[string]multicluster.Aware
}

// New creates a new Multiplexer.
func New() *Multiplexer {
	return &Multiplexer{
		Logger:   mctrl.Log.WithName("multiplexer"),
		Registry: clusters.NewRegistry[cluster.Cluster](),
		awares:   make(map[string]multicluster.Aware),
	}
}

// AddAware adds a multicluster.Aware to the Multiplexer and forwards
// already known cluster.Cluster to it.
func (m *Multiplexer) AddAware(ctx context.Context, name string, aware multicluster.Aware) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	logger := m.Logger.WithValues("aware", name)
	logger.Info("Adding aware")

	m.awares[name] = aware

	return m.Registry.ForEach(func(name multicluster.ClusterName, cl cluster.Cluster) error {
		logger.Info("Engaging aware with existing cluster", "cluster", name)
		return aware.Engage(ctx, name, cl)
	})
}

// DeleteAware deletes a multicluster.Aware from the Multiplexer.
func (m *Multiplexer) DeleteAware(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.awares, name)
}

// Start implements Runnable.
func (m *Multiplexer) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// Engage engages a cluster.Cluster to all multicluster.Aware in the Multiplexer.
func (m *Multiplexer) Engage(ctx context.Context, name multicluster.ClusterName, cl cluster.Cluster) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	logger := m.Logger.WithValues("cluster", name)
	logger.Info("Engaging cluster")

	if err := m.Registry.AddOrReplace(ctx, name, cl); err != nil {
		return fmt.Errorf("error engaging cluster: %w", err)
	}

	for _, aware := range m.awares {
		if err := aware.Engage(ctx, name, cl); err != nil {
			return err
		}
	}

	return nil
}
