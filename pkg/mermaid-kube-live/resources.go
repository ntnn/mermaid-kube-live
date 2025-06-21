package mkl

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
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

func GetResourceStates(ctx context.Context, provider multicluster.Provider, nodes map[string]Node) (map[string]ResourceState, error) {
	ret := make(map[string]ResourceState, len(nodes))

	for name, node := range nodes {
		cluster, err := provider.Get(ctx, node.Selector.Cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster %s: %w", node.Selector.Cluster, err)
		}

		state, err := GetResourceState(ctx, cluster.GetConfig(), node)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource state for node %s: %w", name, err)
		}
		ret[name] = state
	}

	return ret, nil
}

func GetResourceState(ctx context.Context, config *rest.Config, node Node) (ResourceState, error) {
	ret := ResourceState{Status: Absent, Count: 0}

	labelSelector, err := metav1.LabelSelectorAsSelector(&node.Selector.LabelSelector)
	if err != nil {
		return ret, fmt.Errorf("failed to convert label selector: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return ret, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	resources, err := client.
		Resource(node.Selector.GVR).
		Namespace(node.Selector.Namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
	if err != nil {
		return ret, fmt.Errorf("failed to get resource: %w", err)
	}

	ret.Status = Pending

	if node.HealthyWhenPresent && len(resources.Items) > 0 {
		ret.Status = Healthy
	}

	ret.Count = len(resources.Items)
	if allOk(resources.Items, node.HealthType) {
		ret.Status = Healthy
	}

	return ret, nil
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
		condMap, ok := cond.(map[string]interface{})
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
