package mkl

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/multicluster-runtime/providers/file"
)

type GenerateConfig struct {
	Clusters   []string `sep:"," help:"Comma-separated list of clusters to export, all are exported if empty" default:""`
	Scopes     []string `sep:"," short:"s" help:"Comma-separated list of scopes to export" default:"cluster,namespaced"`
	Namespaces []string `sep:"," short:"n" help:"Comma-separated list of namespaces to export, all are exported if empty" default:""`
	// ExportCRDS bool     `help:"Print CRDs as nodes" default:"false"`
	// LabelSelector string `short:"l" help:"Label selector to filter resources" default:""`

	// By default resources only relevant to kuberentes' inner workings
	// and default resources are excluded. ReallyAll allows to include
	// these resources as well.
	// The specific excludes are:
	// - Any GVR defined in DefaultExcludeGVR
	// - Any GVR with a group suffixed with any entry in DefaultExcludeGroups
	// - Any resource with a label matching any entry in DefaultExcludeLabels
	ReallyAll bool `help:"Export _all_ resources, including kubernetes defaults" default:"false"`
}

var (
	// handled when listing apis
	DefaultExcludeGroups = []string{
		"admissionregistration.k8s.io",
		"apiregistration.k8s.io",
		"authentication.k8s.io",
		"authorization.k8s.io",
		"certificates.k8s.io",
		"coordination.k8s.io",
		"events.k8s.io",
		"flowcontrol.apiserver.k8s.io",
		"node.k8s.io",
		"policy",
		"scheduling.k8s.io",
		"storage.k8s.io",
	}
	DefaultExcludeGVK = []metav1.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "Namespace"},
		{Group: "", Version: "v1", Kind: "Node"},
		{Group: "", Version: "v1", Kind: "ComponentStatus"},
	}
	// handled in skipResource
	DefaultExcludeGVKNames = []struct {
		GVK      metav1.GroupVersionKind
		Prefixes []string
		Names    []string
	}{
		{
			GVK: metav1.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
			Prefixes: []string{
				"system:",
				"kubeadm:",
				"local-path-provisioner",
			},
			Names: []string{
				"kindnet",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
			Prefixes: []string{
				"system:",
				"kubeadm:",
				"local-path-provisioner",
			},
			Names: []string{
				"kindnet",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			Names: []string{
				"kube-root-ca.crt",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
			Names: []string{
				"default",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Endpoints"},
			Names: []string{
				"kubernetes",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			Names: []string{
				"kubernetes",
			},
		},
		{
			GVK: metav1.GroupVersionKind{Group: "discovery.k8s.io", Version: "v1", Kind: "EndpointSlice"},
			Names: []string{
				"kubernetes",
			},
		},
	}
	// handled when listing resources
	DefaultExcludeLabels = []string{
		"kubernetes.io/bootstrapping",
	}
	// handled when cycling through namespaces
	DefaultExcludeNamespaces = []string{
		"kube-node-lease",
		"kube-public",
		"kube-system",
		"local-path-storage",
	}
	// handled in skipResource
	DefaultExcludeNamePrefixes = []string{}
)

func skipResource(resource unstructured.Unstructured) bool {
	if resource.GetGenerateName() != "" {
		return true
	}

	for _, prefix := range DefaultExcludeNamePrefixes {
		if strings.HasPrefix(resource.GetName(), prefix) {
			return true
		}
	}

	for _, excludeGVK := range DefaultExcludeGVKNames {
		gvk := resource.GetObjectKind().GroupVersionKind()

		if excludeGVK.GVK.Group != gvk.Group {
			continue
		}
		if excludeGVK.GVK.Version != gvk.Version {
			continue
		}
		if excludeGVK.GVK.Kind != gvk.Kind {
			continue
		}
		for _, prefix := range excludeGVK.Prefixes {
			if strings.HasPrefix(resource.GetName(), prefix) {
				return true
			}
		}
		for _, name := range excludeGVK.Names {
			if resource.GetName() == name {
				return true
			}
		}
	}

	return false
}

func Generate(ctx context.Context, provider *file.Provider, cfg *GenerateConfig) (Config, string, error) {
	retCfg := DefaultConfig()
	retDiagram := strings.Builder{}
	retDiagram.WriteString("flowgraph TD\n")

	clusterNames := provider.ClusterNames()
	if len(cfg.Clusters) > 0 {
		clusterNames = cfg.Clusters
	}

	for _, clusterName := range clusterNames {
		cluster, err := provider.Get(ctx, clusterName)
		if err != nil {
			return retCfg, "", fmt.Errorf("failed to get cluster %s: %w", clusterName, err)
		}

		cl := cluster.GetClient()
		apiResources, err := GatherAPIResources(ctx, cluster.GetConfig(), cfg)
		if err != nil {
			return retCfg, "", fmt.Errorf("failed to gather API resources for cluster %s: %w", clusterName, err)
		}

		retDiagram.WriteString(fmt.Sprintf("  subgraph %s [Cluster: %s]\n", clusterName, clusterName))

		if slices.Contains(cfg.Scopes, "cluster") {
			results, err := GatherResourceLines(ctx, clusterName, cl, apiResources, "", cfg)
			if err != nil {
				return retCfg, "", fmt.Errorf("failed to gather cluster-scoped resources for cluster %s: %w", clusterName, err)
			}

			for _, result := range results {
				retCfg.Nodes[result.NodeName] = result.Node
				retDiagram.WriteString(fmt.Sprintf("    %s\n", result.DiagramLine))
			}
		}

		if slices.Contains(cfg.Scopes, "namespaced") {
			namespaces := cfg.Namespaces
			if len(namespaces) == 0 {
				listedNamespaces, err := Namespaces(ctx, cl, cfg)
				if err != nil {
					return retCfg, "", fmt.Errorf("failed to list namespaces in cluster %s: %w", clusterName, err)
				}
				namespaces = listedNamespaces
			}

			for _, namespace := range namespaces {
				retDiagram.WriteString(fmt.Sprintf("    subgraph %s [Namespace: %s]\n", namespace, namespace))

				results, err := GatherResourceLines(ctx, clusterName, cl, apiResources, namespace, cfg)
				if err != nil {
					return retCfg, "", fmt.Errorf("failed to gather namespaced resources for cluster %s in namespace %s: %w", clusterName, namespace, err)
				}
				for _, result := range results {
					retCfg.Nodes[result.NodeName] = result.Node
					retDiagram.WriteString(fmt.Sprintf("      %s\n", result.DiagramLine))
				}

				retDiagram.WriteString("    end\n")
			}

			retDiagram.WriteString("  end\n")
		}

		retDiagram.WriteString("end\n")
	}

	return retCfg, retDiagram.String(), nil
}

func GatherAPIResources(ctx context.Context, restCfg *rest.Config, cfg *GenerateConfig) ([]metav1.APIResource, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	apiResources := []metav1.APIResource{}
	apiGroupList, err := dcl.ServerGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get server groups: %w", err)
	}

	for _, group := range apiGroupList.Groups {
		skip := false
		for _, excludeGroup := range DefaultExcludeGroups {
			if group.Name == excludeGroup {
				skip = true
				break
			}
		}
		if !cfg.ReallyAll && skip {
			continue
		}

		for _, version := range group.Versions {
			resourceList, err := dcl.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to get server resources for group version %s: %w", version.GroupVersion, err)
			}

			for _, resource := range resourceList.APIResources {
				if resource.Group == "" {
					resource.Group = group.Name
				}
				if resource.Version == "" {
					resource.Version = version.Version
					if resource.Version == "" {
						resource.Version = "v1"
					}
				}
				exclude := false
				for _, excludeGVK := range DefaultExcludeGVK {
					if excludeGVK.Group == resource.Group &&
						excludeGVK.Version == resource.Version &&
						excludeGVK.Kind == resource.Kind {
						exclude = true
						break
					}
				}
				if !cfg.ReallyAll && exclude {
					continue
				}

				apiResources = append(apiResources, resource)
			}
		}
	}

	return apiResources, nil
}

type GatherResourceResult struct {
	NodeName    string
	Node        Node
	DiagramLine string
}

func GatherResourceLines(ctx context.Context, clusterName string, cl client.Client, apiResources []metav1.APIResource, namespace string, cfg *GenerateConfig) ([]GatherResourceResult, error) {
	ret := []GatherResourceResult{}

	for _, resource := range apiResources {
		// Skip namespaced resources when gathering cluster-scoped
		// resources
		if resource.Namespaced && namespace == "" {
			continue
		}
		// And skip cluster-scoped resources when gathering namespaced
		// resources
		if !resource.Namespaced && namespace != "" {
			continue
		}

		// Skip resources that do not have the "list" verb
		if !slices.Contains(resource.Verbs, "list") {
			continue
		}

		gvk := schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}

		list := new(unstructured.UnstructuredList)
		list.SetGroupVersionKind(gvk)

		opts := []client.ListOption{
			NotHasLabels(DefaultExcludeLabels),
		}
		if namespace != "" {
			opts = append(opts, client.InNamespace(namespace))
		}

		if err := cl.List(ctx, list, opts...); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to list resources for %q (GVK): %w", gvk, err)
		}

		for _, r := range list.Items {
			if !cfg.ReallyAll && skipResource(r) {
				continue
			}

			var nodeName string
			if namespace == "" {
				nodeName = fmt.Sprintf("%s-%s-%s", clusterName, resource.Name, r.GetName())
			} else {
				nodeName = fmt.Sprintf("%s-%s-%s-%s", clusterName, namespace, resource.Name, r.GetName())
			}

			ret = append(ret, GatherResourceResult{
				NodeName:    nodeName,
				DiagramLine: fmt.Sprintf("%s[%s: %s]", nodeName, r.GetKind(), r.GetName()),
				Node: Node{
					Selector: NodeSelector{
						Cluster:   clusterName,
						Namespace: namespace,
						GVR: schema.GroupVersionResource{
							Group:    gvk.Group,
							Version:  gvk.Version,
							Resource: resource.Name,
						},
						Name: r.GetName(),
					},
					// TODO make an educated guess about the
					// correct health type
					HealthyWhenPresent: true,
					HealthType:         "",
				},
			})

		}
	}

	return ret, nil
}

func Namespaces(ctx context.Context, cl client.Client, cfg *GenerateConfig) ([]string, error) {
	nsList := &unstructured.UnstructuredList{}
	nsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	})

	if err := cl.List(ctx, nsList, NotHasLabels(DefaultExcludeLabels)); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := []string{}
	for _, ns := range nsList.Items {
		if !cfg.ReallyAll && slices.Contains(DefaultExcludeNamespaces, ns.GetName()) {
			continue
		}
		namespaces = append(namespaces, ns.GetName())
	}

	return namespaces, nil
}
