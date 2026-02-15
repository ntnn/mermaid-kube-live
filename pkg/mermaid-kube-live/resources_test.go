package mkl

import (
	"testing"

	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
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
		node      mklv1alpha1.Node
		expected  ResourceState
	}{
		"configmap by labels": {
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
			node: mklv1alpha1.Node{
				Selector: mklv1alpha1.NodeSelector{
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
				Status: mklv1alpha1.ResourceHealthy,
			},
		},
		"configmap by name": {
			resources: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "configmap-by-name",
						Namespace: namespace,
					},
					Data: map[string]string{},
				},
			},
			node: mklv1alpha1.Node{
				Selector: mklv1alpha1.NodeSelector{
					Namespace: namespace,
					GVR: schema.GroupVersionResource{
						Group:    "",
						Version:  "v1",
						Resource: "configmaps",
					},
					Name: "configmap-by-name",
				},
			},
			expected: ResourceState{
				Count:  1,
				Status: mklv1alpha1.ResourceHealthy,
			},
		},
	}

	cluster, err := provider.Get(t.Context(), clusterName)
	require.NoError(t, err)

	client := cluster.GetClient()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			preCreation, err := GetResourceState(t.Context(), cluster.GetConfig(), tc.node)
			require.NoError(t, err)
			require.Equal(t, ResourceState{Status: mklv1alpha1.ResourceAbsent, Count: 0}, preCreation)

			for _, resource := range tc.resources {
				getOrCreate(t, client, resource)
			}

			postCreation, err := GetResourceState(t.Context(), cluster.GetConfig(), tc.node)
			require.NoError(t, err)
			require.Equal(t, tc.expected, postCreation)
		})
	}
}
