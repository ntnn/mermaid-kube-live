package mkl

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NotHasLabels filters the list/delete operation checking if the set of labels exists
// without checking their values.
type NotHasLabels []string

// ApplyToList applies this configuration to the given list options.
func (m NotHasLabels) ApplyToList(opts *client.ListOptions) {
	if opts.LabelSelector == nil {
		opts.LabelSelector = labels.NewSelector()
	}
	// TODO: ignore invalid labels will result in an empty selector.
	// This is inconsistent to the that of MatchingLabels.
	for _, label := range m {
		r, err := labels.NewRequirement(label, selection.DoesNotExist, nil)
		if err == nil {
			opts.LabelSelector = opts.LabelSelector.Add(*r)
		}
	}
}
