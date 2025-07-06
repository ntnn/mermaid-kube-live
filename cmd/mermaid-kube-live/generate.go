package main

import (
	"context"
	"fmt"
	"log"
	"os"

	yaml "sigs.k8s.io/yaml/goyaml.v2"

	"github.com/ntnn/mermaid-kube-live/pkg/fileprovider"
	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
)

type Generate struct {
	CommonFlags        `embed:""`
	mkl.GenerateConfig `embed:""`
}

func (g *Generate) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("starting kubeconfig provider with files: %s", g.kubeconfig())
	provider, err := fileprovider.FromFiles(g.kubeconfig()...)
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}
	if err := provider.RunOnce(ctx); err != nil {
		return fmt.Errorf("error running provider once: %w", err)
	}

	cfg, diagram, err := mkl.Generate(ctx, provider, &g.GenerateConfig)
	if err != nil {
		return fmt.Errorf("error generating diagram: %w", err)
	}

	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	if err := os.WriteFile(g.Config, cfgBytes, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	if err := os.WriteFile(g.Diagram, []byte(diagram), 0644); err != nil {
		return fmt.Errorf("error writing diagram file: %w", err)
	}

	return nil
}
