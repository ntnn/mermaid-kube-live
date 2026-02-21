package styler

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
	"github.com/ntnn/mermaid-kube-live/pkg/multiplexer"
	mctrl "sigs.k8s.io/multicluster-runtime"
)

// Styler generates styles for the nodes based on the resources
// associated with them.
type Styler struct {
	Logger    logr.Logger
	style     mklv1alpha1.Style
	watches   *watches
	cel       *CELEnv
	resources *resources

	// cached data for each node, keyed by node name in the config
	styleLock sync.RWMutex
	styles    map[string][]string
}

// New creates a new Styler instance.
func New(mp *multiplexer.Multiplexer) (*Styler, error) {
	s := &Styler{}
	s.Logger = mctrl.Log.WithName("styler")
	s.styles = make(map[string][]string)

	celEnv, err := NewCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	s.cel = celEnv

	s.resources = newResources()

	rOpts := reconcilerOpts{
		getCluster:      mp.Registry.Get,
		deleteResource:  s.resources.delete,
		replaceResource: s.resources.replace,
		updateStyling:   s.updateStyling,
	}

	s.watches = newWatches(mp, rOpts)

	return s, nil
}

// UpdateConfig updates the Styler's configuration and refreshes the watches.
func (s *Styler) UpdateConfig(ctx context.Context, config *mklv1alpha1.Config) error {
	s.style = config.Style
	if err := s.watches.update(ctx, config.Nodes); err != nil {
		return fmt.Errorf("failed to update watches: %w", err)
	}

	return nil
}
