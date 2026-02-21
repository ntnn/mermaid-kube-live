package styler

import (
	"slices"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type resources struct {
	lock sync.RWMutex
	res  map[string][]unstructured.Unstructured
}

func newResources() *resources {
	return &resources{
		res: make(map[string][]unstructured.Unstructured),
	}
}

func (r *resources) get(nodeName string) []unstructured.Unstructured {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.res[nodeName]
}

func (r *resources) delete(nodeName string, name, namespace string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.res[nodeName]; !ok {
		return
	}

	r.res[nodeName] = slices.DeleteFunc(r.res[nodeName], func(r unstructured.Unstructured) bool {
		return r.GetName() == name && r.GetNamespace() == namespace
	})
	if len(r.res[nodeName]) == 0 {
		delete(r.res, nodeName)
	}
}

func (r *resources) replace(nodeName string, resource unstructured.Unstructured) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.res[nodeName] = slices.DeleteFunc(r.res[nodeName], func(r unstructured.Unstructured) bool {
		return r.GetName() == resource.GetName() && r.GetNamespace() == resource.GetNamespace()
	})
	r.res[nodeName] = append(r.res[nodeName], resource)
}
