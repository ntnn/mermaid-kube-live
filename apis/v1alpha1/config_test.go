package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	config := &Config{
		Nodes: map[string]Node{
			"node": {
				Selector: NodeSelector{
					ClusterName: "cluster",
					GVR: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					Namespace: "default",
					Name:      "my-deployment",
				},
			},
		},
	}
	require.NoError(t, config.Validate(t.Context()))

	config = &Config{
		Nodes: map[string]Node{
			"node": {
				Selector: NodeSelector{
					ClusterName: "",
					GVR: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					Namespace: "default",
					Name:      "my-deployment",
				},
			},
		},
	}
	require.Error(t, config.Validate(t.Context()))
}
