package fileprovider

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type clusters map[string]cluster.Cluster

func fromContexts(kubeCtxs map[string]*rest.Config) (clusters, error) {
	c := make(map[string]cluster.Cluster, len(kubeCtxs))

	for name, kubeCtx := range kubeCtxs {
		cl, err := cluster.New(kubeCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster for context %q: %w", name, err)
		}
		c[name] = cl
	}

	return c, nil
}

func fromFiles(filepaths ...string) (clusters, error) {
	kubeCtxs := map[string]*rest.Config{}

	for _, filepath := range filepaths {
		fileKubeCtxs, err := ReadContextsFromFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to read kubeconfig from file %q: %w", filepath, err)
		}
		for name, kubeCtx := range fileKubeCtxs {
			if _, exists := kubeCtxs[name]; exists {
				return nil, fmt.Errorf("duplicate context name %q found in file %q", name, filepath)
			}
			kubeCtxs[name] = kubeCtx
		}
	}

	return fromContexts(kubeCtxs)
}

var KubeconfigGlobs = []string{"*.kubeconfig", "*.kubeconfig.yaml", "*.kubeconfig.yml"}

func fromDirectory(dirpath string) (clusters, error) {
	matches := []string{}

	for _, glob := range KubeconfigGlobs {
		globMatches, err := filepath.Glob(filepath.Join(dirpath, glob))
		if err != nil {
			return nil, fmt.Errorf("failed to glob files in directory %q with pattern %q: %w", dirpath, glob, err)
		}
		matches = append(matches, globMatches...)
	}

	return fromFiles(matches...)
}
