package main

import (
	"flag"

	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
)

var (
	fConfig = flag.String("config", "config.yaml", "Path to the configuration file")
)

func getConfig() (*mkl.Config, error) {
	config, err := mkl.ParseFile(*fConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
