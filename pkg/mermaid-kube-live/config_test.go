package mkl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	t.Parallel()
	c, err := ParseFile("../../config-annotated.yaml")
	require.NoError(t, err)
	require.NotNil(t, c)

	require.NotEmpty(t, c.Nodes)

	require.Contains(t, c.Nodes, "NodeName")
	require.Equal(t, c.Nodes["NodeName"].Selector.Namespace, "default")
	require.Contains(t, c.Nodes, "NodeWithCluster")
	require.Equal(t, c.Nodes["NodeWithCluster"].Selector.Cluster, "kind-kind")
}
