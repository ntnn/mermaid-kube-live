// Package mkl combines the other components of mermaid-kube-live to
// provide the main functionality of the project.
package mkl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/ntnn/mcutils"
	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
	"github.com/ntnn/mermaid-kube-live/pkg/multiplexer"
	"github.com/ntnn/mermaid-kube-live/pkg/styler"
	"github.com/ntnn/mermaid-kube-live/pkg/webserver"
	"k8s.io/client-go/rest"
	mctrl "sigs.k8s.io/multicluster-runtime"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

// Options are the options for the MKL instance.
type Options struct {
	// Provider is the multicluster provider of clusters to watch.
	Provider multicluster.Provider

	// ConfigPath is the path to the mermaid-kube-live configuration file.
	ConfigPath string

	// DiagramPath is the path to the mermaid diagram file.
	DiagramPath string

	// UpdateInterval is the interval at which to update the diagram.
	// It not set the diagram will be updated every second.
	UpdateInterval time.Duration

	// Adresss is the address of the webserver.
	Address string

	// Logger is the logger to use.
	Logger logr.Logger
}

// FlagSet returns a flag set for the options.
func (o *Options) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("mkl", flag.ExitOnError)

	fs.StringVar(&o.ConfigPath, "config", "", "Configuration file")
	fs.StringVar(&o.DiagramPath, "diagram", "", "Diagram file")
	fs.DurationVar(&o.UpdateInterval, "update-interval", time.Second, "Interval to update the diagram")
	fs.StringVar(&o.Address, "address", "localhost:8080", "Address to listen on")

	return fs
}

// Validate validates the options.
func (o *Options) Validate() error {
	if o.Provider == nil {
		return errors.New("provider is required")
	}

	if o.ConfigPath == "" {
		return errors.New("config path is required")
	}

	if o.DiagramPath == "" {
		return errors.New("diagram path is required")
	}

	if o.UpdateInterval <= 0 {
		o.UpdateInterval = time.Second
	}

	if o.Address == "" {
		o.Address = "localhost:8080"
	}

	if o.Logger.GetSink() == nil {
		o.Logger = logr.Discard()
	}

	return nil
}

// MKL is the main struct for the mermaid-kube-live application.
type MKL struct {
	opts *Options

	web    *webserver.WebServer
	styler *styler.Styler

	diagramLock sync.RWMutex
	diagram     []byte
}

// New creates a new MKL instance with the given options.
func New(opts *Options) (*MKL, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	instance := new(MKL)
	instance.opts = opts

	return instance, nil
}

// Run starts the MKL instance and blocks until the context is canceled
// or an error occurs.
func (m *MKL) Run(ctx context.Context) error {
	if err := m.startWebServer(ctx); err != nil {
		return fmt.Errorf("error starting web server: %w", err)
	}

	if err := m.watchDiagram(ctx); err != nil {
		return fmt.Errorf("error watching diagram file: %w", err)
	}

	if err := m.startStyler(ctx); err != nil {
		return fmt.Errorf("error starting styler: %w", err)
	}

	if err := m.watchConfig(ctx); err != nil {
		return fmt.Errorf("error watching config file: %w", err)
	}

	// TODO instead of updating the diagram on a fixed interval, it
	// should update it whenever the styler updates
	// already got handlers for diagram changes, just need another one
	// for styler changes.
	for range time.Tick(m.opts.UpdateInterval) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		styling, err := m.styler.GetStyling()
		if err != nil {
			m.opts.Logger.Error(err, "failed to get styling")
			continue
		}

		b := strings.Builder{}

		m.diagramLock.RLock()
		b.Write(m.diagram)
		m.diagramLock.RUnlock()

		b.WriteString("\n")
		b.WriteString(styling)

		m.web.UpdateDiagram([]byte(b.String()))
		// m.opts.Logger.V(2).Info("diagram updated", "content", b.String())
	}

	return nil
}

func (m *MKL) startWebServer(ctx context.Context) error {
	m.web = &webserver.WebServer{
		Logger: m.opts.Logger.WithName("webserver"),
	}

	if err := m.web.Start(ctx, m.opts.Address); err != nil {
		return fmt.Errorf("error starting web server: %w", err)
	}
	m.opts.Logger.Info("web server started", "addr", m.web.Server.Addr)
	return nil
}

func (m *MKL) startStyler(ctx context.Context) error {
	mgr, err := mctrl.NewManager(&rest.Config{}, m.opts.Provider, mcutils.SilentManagerOpts(mctrl.Options{
		Logger: m.opts.Logger.WithName("multicluster-manager"),
	}))
	if err != nil {
		return fmt.Errorf("error creating multicluster manager: %w", err)
	}

	mp := multiplexer.New()
	if err := mgr.Add(mp); err != nil {
		return fmt.Errorf("error adding multiplexer to manager: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			m.opts.Logger.Error(err, "multicluster manager errored")
		}
	}()

	st, err := styler.New(mp)
	if err != nil {
		return fmt.Errorf("failed to create styler: %w", err)
	}
	m.styler = st
	return nil
}

func (m *MKL) watchDiagram(ctx context.Context) error {
	return m.watchFile(ctx, m.opts.DiagramPath, func() error {
		rawDiagram, err := os.ReadFile(m.opts.DiagramPath)
		if err != nil {
			return fmt.Errorf("failed to read diagram file %s: %w", m.opts.DiagramPath, err)
		}

		m.diagramLock.Lock()
		m.diagram = rawDiagram
		m.diagramLock.Unlock()
		m.opts.Logger.V(2).Info("diagram file updated", "file", m.opts.DiagramPath, "content", string(rawDiagram))

		return nil
	})
}

func (m *MKL) watchConfig(ctx context.Context) error {
	if m.styler == nil {
		return errors.New("styler is not initialized")
	}
	return m.watchFile(ctx, m.opts.ConfigPath, func() error {
		config, err := mklv1alpha1.ParseFile(m.opts.ConfigPath)
		if err != nil {
			return fmt.Errorf("failed to parse config file %s: %w", m.opts.ConfigPath, err)
		}

		if err := config.Validate(ctx); err != nil {
			return fmt.Errorf("invalid config file %s: %w", m.opts.ConfigPath, err)
		}

		if err := m.styler.UpdateConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}

		m.opts.Logger.V(2).Info("config file updated", "file", m.opts.ConfigPath, "content", config)

		return nil
	})
}
