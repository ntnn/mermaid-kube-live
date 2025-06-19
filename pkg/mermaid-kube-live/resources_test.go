package mermaidkubelive

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetResourceState(t *testing.T) {
	cases := map[string]struct {
		node     Node
		expected ResourceState
	}{
		"configmap": {
			node: Node{
				HealthyWhenPresent: true,
				Selector: NodeSelector{
					Namespace: "resources",
					GVR: schema.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "configmaps",
					},
					LabelSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testname": "configmap",
						},
					},
				},
			},
			expected: ResourceState{
				Count:  1,
				Status: Healthy,
			},
		},
	}

	cluster, err := provider.Get(t.Context(), "kind-mkl-one")
	require.NoError(t, err)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			state, err := GetResourceState(t.Context(), cluster.GetConfig(), tc.node)
			require.NoError(t, err)
			require.Equal(t, tc.expected, state)
		})
	}
}
