package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	t.Parallel()
	c, err := ParseFile("example.yaml")
	require.NoError(t, err)
	require.NotNil(t, c)

	require.NotEmpty(t, c.Nodes)

	require.Contains(t, c.Nodes, "node1")
	require.Equal(t, c.Nodes["node1"].Selector.Namespace, "default")
	require.Contains(t, c.Nodes, "node2")
	require.Equal(t, c.Nodes["node2"].Selector.ClusterName, "./kubeconfig+kind-kind")
}
