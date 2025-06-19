package fileprovider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromFiles_Single(t *testing.T) {
	t.Parallel()
	provider, err := FromFiles("testdata/single.kubeconfig.yaml")
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 1)
	require.Contains(t, provider.clusters, "local")
}

func TestFromFiles_Multi(t *testing.T) {
	t.Parallel()
	provider, err := FromFiles("testdata/multi.kubeconfig.yaml")
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 3)
	require.Contains(t, provider.clusters, "one")
	require.Contains(t, provider.clusters, "two")
	require.Contains(t, provider.clusters, "three")
}

func TestFromFiles_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	provider, err := FromFiles("../../integration.kubeconfig.yaml")
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 2)
	require.Contains(t, provider.clusters, "kind-mkl-one")
	require.Contains(t, provider.clusters, "kind-mkl-two")
}

func TestFromDirectory(t *testing.T) {
	t.Parallel()
	provider, err := FromDirectory("testdata")
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 4)
	require.Contains(t, provider.clusters, "local")
	require.Contains(t, provider.clusters, "one")
	require.Contains(t, provider.clusters, "two")
	require.Contains(t, provider.clusters, "three")
}

func TestFromDirectory_Integration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	provider, err := FromDirectory("../../")
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 2)
	require.Contains(t, provider.clusters, "kind-mkl-one")
	require.Contains(t, provider.clusters, "kind-mkl-two")
}
