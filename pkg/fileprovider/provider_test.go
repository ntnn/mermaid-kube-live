package fileprovider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromFiles_Single(t *testing.T) {
	t.Parallel()
	provider, err := FromFiles("testdata/single.kubeconfig.yaml")
	require.NoError(t, provider.RunOnce(t.Context()))
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 1)
	require.Contains(t, provider.clusters, "local")
}

func TestFromFiles_Multi(t *testing.T) {
	t.Parallel()
	provider, err := FromFiles("testdata/multi.kubeconfig.yaml")
	require.NoError(t, provider.RunOnce(t.Context()))
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 3)
	require.Contains(t, provider.clusters, "one")
	require.Contains(t, provider.clusters, "two")
	require.Contains(t, provider.clusters, "three")
}

func TestFromDirectory(t *testing.T) {
	t.Parallel()
	provider, err := FromDirectory("testdata")
	require.NoError(t, provider.RunOnce(t.Context()))
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Len(t, provider.clusters, 4)
	require.Contains(t, provider.clusters, "local")
	require.Contains(t, provider.clusters, "one")
	require.Contains(t, provider.clusters, "two")
	require.Contains(t, provider.clusters, "three")
}
