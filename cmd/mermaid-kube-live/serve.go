package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
	"sigs.k8s.io/multicluster-runtime/providers/file"
)

//go:embed serve.html
var mainPage string

type Serve struct {
	CommonFlags `embed:""`

	builtDiagram     string
	builtDiagramLock sync.Mutex

	notifyChan chan struct{}

	Host string `help:"Host to listen on" default:"localhost"`
	Port int    `help:"Port to listen on" default:"8080"`

	UpdateInterval time.Duration `short:"i" help:"Interval to update the diagram" default:"1s"`
}

func (s *Serve) Run() error {
	ctx := context.Background()

	// Serve the main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(mainPage)); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})

	// Serve the built diagram
	http.HandleFunc("/diagram", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		s.builtDiagramLock.Lock()
		ret := []byte(s.builtDiagram)
		s.builtDiagramLock.Unlock()
		if _, err := w.Write(ret); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})

	// Event loop to notify clients about diagram updates
	s.notifyChan = make(chan struct{}, 10)
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		defer r.Context().Done()
		for range s.notifyChan {
			if _, err := fmt.Fprintf(w, "data: diagram updated\n\n"); err != nil {
				log.Printf("failed to write to response: %v", err)
				return
			}
			w.(http.Flusher).Flush()
		}
	})

	log.Printf("starting webserver on %s:%d", s.Host, s.Port)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%s:%d", s.Host, s.Port), nil); err != nil {
			log.Printf("error from the web server: %v", err)
		}
	}()

	log.Printf("starting kubeconfig provider with files: %s", s.kubeconfig())

	provider, err := file.New(file.Options{
		KubeconfigFiles: s.kubeconfig(),
	})
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}
	if err := provider.RunOnce(ctx, nil); err != nil {
		return fmt.Errorf("error running provider once: %w", err)
	}
	go func() {
		if err := provider.Run(ctx, nil); err != nil {
			log.Printf("provider errored: %v", err)
		}
	}()

	for range time.Tick(s.UpdateInterval) {
		rawDiagram, err := os.ReadFile(s.Diagram)
		if err != nil {
			log.Printf("failed to read diagram file %s: %v", s.Diagram, err)
			continue
		}
		diagram := string(rawDiagram) + "\n"

		config, err := mkl.ParseFile(s.Config)
		if err != nil {
			log.Printf("failed to read config file %s: %v", s.Config, err)
			continue
		}

		nodeStates, err := mkl.GetResourceStates(ctx, provider, config.Nodes)
		if err != nil {
			log.Printf("failed to get resource states, skipping update: %v", err)
			continue
		}

		for name, state := range nodeStates {
			style, ok := config.StatusStyle[state.Status]
			if !ok {
				log.Printf("unknown status %s for node %s, skipping", state.Status, name)
				continue
			}
			diagram += fmt.Sprintf("style %s %s\n", name, style)
		}

		if s.builtDiagram == diagram {
			log.Println("diagram is unchanged, skipping update")
			continue
		}

		s.builtDiagramLock.Lock()
		s.builtDiagram = diagram
		s.builtDiagramLock.Unlock()

		log.Println("diagram updated successfully, notifying clients")
		s.notifyChan <- struct{}{}
	}

	return nil
}
