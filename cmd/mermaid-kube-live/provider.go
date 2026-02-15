package main

import (
	"context"
	"fmt"
	"log"

	"sigs.k8s.io/multicluster-runtime/providers/file"
)

func (s *Serve) startProvider(ctx context.Context) error {
	log.Printf("starting kubeconfig provider with files: %s", s.kubeconfig())

	provider, err := file.New(file.Options{
		KubeconfigFiles: s.kubeconfig(),
	})
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	s.provider = provider

	if err := provider.RunOnce(ctx, nil); err != nil {
		return fmt.Errorf("error running provider once: %w", err)
	}

	// TODO if the provider fails the whole program should exit.
	go func() {
		if err := provider.Start(ctx, nil); err != nil {
			log.Printf("provider errored: %v", err)
		}
	}()

	return nil
}
