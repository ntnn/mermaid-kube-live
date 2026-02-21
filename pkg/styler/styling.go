package styler

import (
	"context"
	"fmt"
	"strings"

	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
)

// GetStyling returns the current styles for all nodes.
func (s *Styler) GetStyling() (string, error) {
	var ret strings.Builder

	s.styleLock.RLock()
	defer s.styleLock.RUnlock()

	for _, styles := range s.styles {
		for _, style := range styles {
			ret.WriteString(style)
		}
	}

	return ret.String(), nil
}

func (s *Styler) updateStyling(ctx context.Context, nodeName string, node mklv1alpha1.Node) error {
	logger := s.Logger.WithValues("nodeName", nodeName)
	logger.V(2).Info("updating styling for node")

	resources := s.resources.get(nodeName)

	status := resourceStatus(node, resources)
	s.Logger.V(2).Info("updated status", "status", status, "resourceCount", len(resources))

	newStyles := []string{}

	style, ok := s.style.Status[status]
	if !ok {
		style = status.DefaultStyle()
		if style == "" {
			logger.Error(nil, "unknown status and no default style available, skipping styling update", "status", status)
		}
	}

	newStyles = append(newStyles, fmt.Sprintf("style %s %s\n", nodeName, style))

	if node.Label != "" {
		logger.V(2).Info("expanding label", "label", node.Label)
		label, err := s.cel.expandLabel(ctx, node.Label, resources)
		if err != nil {
			logger.Error(err, "failed to expand label, skipping label update", "label", node.Label)
		} else {
			logger.V(2).Info("expanded label", "label", node.Label, "expanded", label)
			newStyles = append(newStyles, fmt.Sprintf("%s[%s]\n", nodeName, label))
		}
	}

	s.styleLock.Lock()
	s.styles[nodeName] = newStyles
	s.styleLock.Unlock()

	return nil
}
