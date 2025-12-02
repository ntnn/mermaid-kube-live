package v1alpha1

// ResourceStatus represents the status of a resource in a cluster.
type ResourceStatus string

const (
	// ResourceAbsent indicates that the resource is not present.
	ResourceAbsent ResourceStatus = "absent"
	// ResourcePending indicates that the resource is present but not healthy.
	ResourcePending ResourceStatus = "pending"
	// ResourceHealthy indicates that the resource is present and healthy.
	ResourceHealthy ResourceStatus = "healthy"
)

// String returns the string representation of the ResourceStatus.
func (rs ResourceStatus) String() string {
	return string(rs)
}

// DefaultStyle returns the default style for the ResourceStatus.
func (rs ResourceStatus) DefaultStyle() string {
	switch rs {
	case ResourceAbsent:
		return "stroke:#808080"
	case ResourcePending:
		return "stroke:#FFFF00"
	case ResourceHealthy:
		return "stroke:#00FF00"
	default:
		return ""
	}
}
