package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	_ "embed"
)

//go:embed serve.html
var mainPage string

type Serve struct {
	CommonFlags `embed:""`

	builtDiagram     string
	builtDiagramLock sync.Mutex

	notifyChan chan struct{}

	Host string `default:"localhost" help:"Host to listen on"`
	Port int    `default:"8080"      help:"Port to listen on"`

	provider multicluster.Provider

	UpdateInterval time.Duration `default:"1s" help:"Interval to update the diagram" short:"i"`
}

func (s *Serve) Run() error {
	ctx := context.Background()

	s.httpServer(ctx)

	if err := s.startProvider(ctx); err != nil {
		return fmt.Errorf("failed to start provider: %w", err)
	}

	for range time.Tick(s.UpdateInterval) {
		if err := s.iteration(ctx); err != nil {
			log.Printf("%v", err)
		}
	}

	return nil
}

func (s *Serve) iteration(ctx context.Context) error { //nolint:cyclop
	rawDiagram, err := os.ReadFile(s.Diagram)
	if err != nil {
		return fmt.Errorf("failed to read diagram file %s: %w", s.Diagram, err)
	}

	diagram := string(rawDiagram) + "\n"

	config, err := mklv1alpha1.ParseFile(s.Config)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", s.Config, err)
	}

	if err := config.Validate(ctx); err != nil {
		return fmt.Errorf("invalid config file %s: %w", s.Config, err)
	}

	nodeStates, err := mkl.GetResourceStates(ctx, s.provider, config.Nodes)
	if err != nil {
		return fmt.Errorf("failed to get resource states, skipping update: %w", err)
	}

	var diagramSb126 strings.Builder

	for name, state := range nodeStates {
		style, ok := config.Style.Status[state.Status]
		if !ok {
			style = state.Status.DefaultStyle()
			if style == "" {
				// TODO maybe a fallback style, e.g. red outline for
				// unknown status?
				return fmt.Errorf("unknown status %s for node %s and no default style available, skippinw", state.Status, name)
			}
		}

		diagramSb126.WriteString(fmt.Sprintf("style %s %s\n", name, style))

		if configLabel := config.Nodes[name].Label; configLabel != "" {
			label, err := expandLabel(ctx, configLabel, state)
			if err != nil {
				return fmt.Errorf("failed to expand label for node %s, skipping label update: %w", name, err)
			}

			diagramSb126.WriteString(fmt.Sprintf("%s[%s]\n", name, label))
		}
	}

	diagram += diagramSb126.String()

	if s.builtDiagram == diagram {
		return errors.New("diagram is unchanged, skipping updatw")
	}

	s.builtDiagramLock.Lock()
	s.builtDiagram = diagram
	s.builtDiagramLock.Unlock()

	log.Println("diagram updated successfully, notifying clients")

	s.notifyChan <- struct{}{}

	return nil
}
