package mkl

import (
	"context"
	"fmt"
	"log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
)

type ResourceState struct {
	Resources []map[string]any           `json:"resources,omitempty"`
	Status    mklv1alpha1.ResourceStatus `json:"status"`
	Count     int                        `json:"count"`
}

func GetResourceStates(ctx context.Context, provider multicluster.Provider, nodes map[string]mklv1alpha1.Node) (map[string]ResourceState, error) {
	ret := make(map[string]ResourceState, len(nodes))

	for name, node := range nodes {
		cluster, err := provider.Get(ctx, node.Selector.ClusterName)
		if err != nil {
			log.Printf("failed to get cluster %s, setting node absent: %v", node.Selector.ClusterName, err)
			ret[name] = ResourceState{Status: mklv1alpha1.ResourceAbsent, Count: 0}
			continue
		}

		state, err := GetResourceState(ctx, cluster.GetConfig(), node)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource state for node %s: %w", name, err)
		}
		ret[name] = state
	}

	return ret, nil
}

func GetResourceState(ctx context.Context, config *rest.Config, node mklv1alpha1.Node) (ResourceState, error) {
	ret := ResourceState{
		Status: mklv1alpha1.ResourceAbsent,
		Count:  0,
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return ret, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	var resources []unstructured.Unstructured

	switch {
	case node.Selector.Name != "":
		resourceByName, err := client.Resource(node.Selector.GVR).
			Namespace(node.Selector.Namespace).
			Get(ctx, node.Selector.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return ret, nil // resource not found, return absent state
		}
		if err != nil {
			return ret, fmt.Errorf("failed to get resource by name: %w", err)
		}
		resources = []unstructured.Unstructured{*resourceByName}
	case node.Selector.Owner.Name != "":
		ownerByName, err := client.Resource(node.Selector.Owner.GVR).
			Namespace(node.Selector.Namespace).
			Get(ctx, node.Selector.Owner.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return ret, nil // owner resource not found, return absent state
		}
		if err != nil {
			return ret, fmt.Errorf("failed to get owner resource: %w", err)
		}

		ownerUID := ownerByName.GetUID()
		resourceByOwner, err := client.
			Resource(node.Selector.GVR).
			Namespace(node.Selector.Namespace).
			List(ctx, metav1.ListOptions{})
		if apierrors.IsNotFound(err) || resourceByOwner == nil || len(resourceByOwner.Items) == 0 {
			return ret, nil // no resources found, return absent state
		}
		if err != nil {
			return ret, fmt.Errorf("failed to get resource by owner: %w", err)
		}

		var ownedResources []unstructured.Unstructured
		for _, item := range resourceByOwner.Items {
			owners := item.GetOwnerReferences()
			for _, owner := range owners {
				if owner.UID == ownerUID {
					ownedResources = append(ownedResources, item)
					break
				}
			}
		}
		if len(ownedResources) == 0 {
			return ret, nil // no owned resources found, return absent state
		}
		resources = ownedResources
	default:
		labelSelector, err := metav1.LabelSelectorAsSelector(&node.Selector.LabelSelector)
		if err != nil {
			return ret, fmt.Errorf("failed to convert label selector: %w", err)
		}

		resourceByLabels, err := client.
			Resource(node.Selector.GVR).
			Namespace(node.Selector.Namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector.String(),
			})
		if apierrors.IsNotFound(err) || resourceByLabels == nil || len(resourceByLabels.Items) == 0 {
			return ret, nil // no resources found, return absent state
		}
		if err != nil {
			return ret, fmt.Errorf("failed to get resource: %w", err)
		}

		resources = resourceByLabels.Items
	}

	ret.Status = mklv1alpha1.ResourcePending
	if node.Health.WhenPresent && len(ret.Resources) > 0 {
		ret.Status = mklv1alpha1.ResourceHealthy
	}

	ret.Count = len(ret.Resources)
	if allOk(resources, node.Health.ConditionType) {
		ret.Status = mklv1alpha1.ResourceHealthy
	}

	for _, res := range resources {
		ret.Resources = append(ret.Resources, res.Object)
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
