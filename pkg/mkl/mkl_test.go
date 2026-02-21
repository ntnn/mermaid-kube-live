package mkl

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ntnn/mcutils/mctest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
	clustersprovider "sigs.k8s.io/multicluster-runtime/providers/clusters"
)

func filterContextCancelled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func TestMKL(t *testing.T) {
	t.Parallel()

	env := mctest.EnvTest(t, nil)

	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, nil, 0600))

	diagramPath := filepath.Join(tempDir, "diagram.mermaid")
	require.NoError(t, os.WriteFile(diagramPath, nil, 0600))

	cls := clustersprovider.New()
	opts := &Options{
		Provider:    cls,
		ConfigPath:  configPath,
		DiagramPath: diagramPath,
	}

	mkl, err := New(opts)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	wg := &sync.WaitGroup{}
	wg.Go(func() {
		assert.NoError(t, filterContextCancelled(mkl.Run(ctx)))
	})

	cl, err := cluster.New(env.Config)
	require.NoError(t, err)
	require.NoError(t, cls.Add(t.Context(), multicluster.ClusterName("envtest"), cl))

	require.Eventually(t, func() bool {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost:8080/diagram", nil)
		require.NoError(t, err) // Can use require here because this should never fail.
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}
		if err := resp.Body.Close(); err != nil {
			return false
		}
		return resp.StatusCode == http.StatusOK
	}, wait.ForeverTestTimeout, time.Second)

	// Cancel the context to stop MKL and wait for the graceful finish.
	cancel()
	wg.Wait()
}
