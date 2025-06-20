package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
)

var (
	// The path to the diagram file to be served
	fDiagram = flag.String("diagram", "", "Path to the diagram file to be served")
	// The raw diagram content read from the file
	rawDiagram string
	// The diagram that will be updated
	diagram string
)

func init() {
	http.HandleFunc("/diagram", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(diagram)); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})
}

func loadDiagram() error {
	if *fDiagram == "" {
		return fmt.Errorf("diagram file path is required")
	}

	data, err := os.ReadFile(*fDiagram)
	if err != nil {
		return fmt.Errorf("failed to read diagram file: %w", err)
	}

	rawDiagram = string(data)
	diagram = rawDiagram
	notifyChan <- struct{}{}
	return nil
}

func updateDiagramLoop(ctx context.Context) error {
	if err := loadDiagram(); err != nil {
		return fmt.Errorf("failed to load diagram: %w", err)
	}

	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	config, err := getConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	updateInterval, err := time.ParseDuration(config.UpdateInterval)
	if err != nil {
		return fmt.Errorf("invalid update interval %q: %w", config.UpdateInterval, err)
	}

	go func() {
		for range time.Tick(updateInterval) {
			if ctx.Err() != nil {
				log.Println("context cancelled, stopping update loop")
				return
			}

			log.Printf("updating diagram with nodes\n")

			nodeStates, err := mkl.GetResourceStates(ctx, provider, config.Nodes)
			if err != nil {
				log.Printf("failed to get resource states, skipping update: %v", err)
				continue
			}

			newDiagram := rawDiagram + "\n"
			for name, state := range nodeStates {
				style, ok := config.StatusStyle[state.Status]
				if !ok {
					log.Printf("unknown status %s for node %s, skipping", state.Status, name)
					continue
				}
				newDiagram += fmt.Sprintf("style %s %s\n", name, style)
			}

			log.Println("diagram updated successfully")
			diagram = newDiagram
			notifyChan <- struct{}{}
		}
	}()

	return nil
}
