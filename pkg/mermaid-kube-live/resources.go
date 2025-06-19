package mermaidkubelive

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type ResourceStatus string

const (
	Absent  ResourceStatus = "absent"
	Pending ResourceStatus = "pending"
	Healthy ResourceStatus = "healthy"
)

type ResourceState struct {
	Status ResourceStatus `json:"status"`
	Count  int            `json:"count"`
}

func GetResourceState(ctx context.Context, config *rest.Config, node Node) (ResourceState, error) {
	ret := ResourceState{Status: Absent, Count: 0}

	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&node.Selector.LabelSelector),
	}

	resources, err := dynamic.NewForConfigOrDie(config).
		Resource(node.Selector.GVR).
		Namespace(node.Selector.Namespace).
		List(ctx, listOptions)
	if err != nil {
		return ret, fmt.Errorf("failed to get resource: %w", err)
	}

	ret.Status = Pending

	if node.HealthyWhenPresent && len(resources.Items) > 0 {
		ret.Status = Healthy
	}

	ret.Count = len(resources.Items)

	for _, item := range resources.Items {
		status, found, err := unstructured.NestedSlice(item.Object, "status", "conditions")
		if err != nil || !found {
			continue
		}

		for _, condition := range status {
			conditionMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}
			if conditionMap["type"] == node.HealthType && conditionMap["status"] != metav1.ConditionTrue {
				return ret, nil
			}
		}
	}

	ret.Status = Healthy
	return ret, nil
}
