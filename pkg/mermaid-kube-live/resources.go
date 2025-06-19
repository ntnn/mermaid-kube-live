package mermaidkubelive

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ret.Count = len(resources.Items)

	// TODO check status of resources and set to healthy if all are
	// healthy

	return ret, nil
}
