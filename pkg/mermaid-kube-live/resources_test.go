package mkl

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetResourceState(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	namespace := namespace(t)

	cases := map[string]struct {
		resources []client.Object
		node      Node
		expected  ResourceState
	}{
		"configmap": {
			resources: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "configmap",
						Namespace: namespace,
						Labels: map[string]string{
							"testname": "configmap",
						},
					},
					Data: map[string]string{},
				},
			},
			node: Node{
				HealthyWhenPresent: true,
				Selector: NodeSelector{
					Namespace: namespace,
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

	cluster, err := provider.Get(t.Context(), clusterName)
	require.NoError(t, err)

	client := cluster.GetClient()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, resource := range tc.resources {
				getOrCreate(t, client, resource)
			}

			state, err := GetResourceState(t.Context(), cluster.GetConfig(), tc.node)
			require.NoError(t, err)
			require.Equal(t, tc.expected, state)
		})
	}
}
