package styler

import (
	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func resourceStatus(node mklv1alpha1.Node, resources []unstructured.Unstructured) mklv1alpha1.ResourceStatus {
	if len(resources) == 0 {
		return mklv1alpha1.ResourceAbsent
	}

	if node.Health.WhenPresent && len(resources) > 0 {
		return mklv1alpha1.ResourceHealthy
	}

	if allOk(resources, node.Health.ConditionType) {
		return mklv1alpha1.ResourceHealthy
	}

	return mklv1alpha1.ResourcePending
}

func allOk(items []unstructured.Unstructured, healthType string) bool {
	for _, item := range items {
		status, found, err := unstructured.NestedSlice(item.Object, "status", "conditions")
		if err != nil || !found {
			continue
		}

		if !statusOk(status, healthType) {
			return false
		}
	}

	return true
}

func statusOk(status []any, healthType string) bool {
	for _, cond := range status {
		condMap, ok := cond.(map[string]any)
		if !ok {
			continue
		}

		if condMap["type"] != healthType {
			continue
		}

		return condMap["status"] == string(metav1.ConditionTrue)
	}
	// default to ok if the condition type is not found
	return true
}
