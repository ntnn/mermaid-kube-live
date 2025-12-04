package v1alpha1

import (
	context "context"

	operation "k8s.io/apimachinery/pkg/api/operation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	field "k8s.io/apimachinery/pkg/util/validation/field"
)

// Config is the configuration for mermaid-kube-live.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Config struct {
	metav1.TypeMeta `json:",inline"`

	// Style defines base values for dynamic styling of the diagram.
	Style Style `json:"style,omitempty"`

	// Nodes is a map of node names to their configuration.
	Nodes map[string]Node `json:"nodes,omitempty"`
}

// Validate validates the Config object.
func (c *Config) Validate(ctx context.Context) error {
	fieldErrors := Validate_Config(
		ctx,
		operation.Operation{
			Type:    operation.Create,
			Request: operation.Request{},
			Options: []string{},
		},
		&field.Path{},
		c,
		c,
	)
	if len(fieldErrors) > 0 {
		return fieldErrors.ToAggregate()
	}
	// TODO custom validation for e.g. node selectors
	return nil
}

// Style defines styling options for the diagram.
type Style struct {
	// Status defines styles for different resource statuses.
	Status map[ResourceStatus]string `json:"status,omitempty"`
}

// Node represents a node in the diagram.
type Node struct {
	// Selector defines how to select the resources for this node.
	Selector NodeSelector `json:"selector"`

	// Health defines how to determine the health of the node.
	Health Health `json:"health,omitempty"`

	// Label is an optional label to display for the node.
	// This is a CEL expression.
	// The input is the ResourceState object for the node, named `rs`.
	Label string `json:"label,omitempty"`
}

// NodeSelector defines how to select resources in a cluster.
type NodeSelector struct {
	// ClusterName is the name of the cluster to select resources from.
	//+k8s:required
	ClusterName string `json:"clusterName"`

	// GVR is the GroupVersionResource of the resources to select.
	GVR schema.GroupVersionResource `json:"gvr"`

	// Name is the name of the resource to select.
	Name string `json:"name,omitempty"`
	// Namespace is the namespace of the resources to select.
	Namespace string `json:"namespace,omitempty"`

	// LabelSelector is the label selector to select resources.
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`

	// If set, select resources owned by the specified owner.
	// This is still bound by the GVR and Namespace fields.
	Owner OwnerReference `json:"owner,omitempty"`
}

// OwnerReference defines an owner resource to select by.
type OwnerReference struct {
	// GVR is the GroupVersionResource of the owner.
	GVR schema.GroupVersionResource `json:"gvr,omitempty"`
	// Name is the name of the owner resource.
	Name string `json:"name,omitempty"`
}

// Health defines how to determine the health of a resource.
type Health struct {
	// WhenPresent indicates if the resource is healthy when present.
	// This is the default when no other option is set.
	WhenPresent bool `json:"whenPresent,omitempty"`

	// ConditionType is the condition type to check for health.
	// If set, the resource is healthy when the condition of this type is True.
	ConditionType string `json:"conditionType,omitempty"`
}
