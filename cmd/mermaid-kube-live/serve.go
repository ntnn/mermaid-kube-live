package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ntnn/mermaid-kube-live/pkg/fileprovider"
	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
)

//go:embed serve.html
var mainPage string

type Serve struct {
	ConfigFile      string `short:"c" long:"config" help:"Path to the configuration file" default:"config.yaml"`
	DiagramFile     string `short:"d" long:"diagram" help:"Path to the diagram file" required:"true"`
	KubeconfigFiles string `short:"k" long:"kubeconfig-files" help:"Comma-separated list of kubeconfig files to use for clusters" required:"true"`

	builtDiagram     string
	builtDiagramLock sync.Mutex

	notifyChan chan struct{}

	Host string `short:"H" long:"host" help:"Host to listen on" default:"localhost"`
	Port int    `short:"p" long:"port" help:"Port to listen on" default:"8080"`

	UpdateInterval time.Duration `short:"i" long:"update-interval" help:"Interval to update the diagram" default:"1s"`
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

	// updateInterval, err := time.ParseDuration(*fUpdateInterval)
	// if err != nil {
	// 	return fmt.Errorf("invalid update interval %q: %w", *fUpdateInterval, err)
	// }

	log.Printf("starting kubeconfig provider with files: %s", s.KubeconfigFiles)
	provider, err := fileprovider.FromFiles(strings.Split(s.KubeconfigFiles, ",")...)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}
	go func() {
		if err := provider.Run(ctx); err != nil {
			log.Printf("failed to run provider: %v", err)
		}
	}()

	for range time.Tick(s.UpdateInterval) {
		rawDiagram, err := os.ReadFile(s.DiagramFile)
		if err != nil {
			log.Printf("failed to read diagram file %s: %v", s.DiagramFile, err)
			continue
		}
		diagram := string(rawDiagram) + "\n"

		config, err := mkl.ParseFile(s.ConfigFile)
		if err != nil {
			log.Printf("failed to read config file %s: %v", s.ConfigFile, err)
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
