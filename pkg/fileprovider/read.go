package fileprovider

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func ReadContextsFromFile(filepath string) (map[string]*rest.Config, error) {
	config, err := clientcmd.LoadFromFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from file %q: %w", filepath, err)
	}

	ret := make(map[string]*rest.Config, len(config.Contexts))
	for name := range config.Contexts {
		restConfig, err := clientcmd.NewNonInteractiveClientConfig(*config, name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create rest config for context %q: %w", name, err)
		}
		ret[name] = restConfig
	}

	return ret, nil
}
