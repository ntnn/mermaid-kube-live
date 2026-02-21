// Package webserver is the webserver for mermaid-kube-live serving the diagram on a web page.
package webserver

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

const (
	readTimeout     = 5 * time.Minute
	shutdownTimeout = 5 * time.Second
)

// WebServer is a web server that serves the diagram on a web page and notifies clients about diagram updates.
type WebServer struct {
	Server *http.Server
	Logger logr.Logger

	// notifyChan is used to notify clients about diagram updates.
	notifyChan chan struct{}

	// diagram is the current diagram to serve.
	diagramLock sync.RWMutex
	diagram     []byte
}

// UpdateDiagram updates the diagram to serve and notifies clients about the update.
func (s *WebServer) UpdateDiagram(diagram []byte) {
	s.diagramLock.Lock()
	s.diagram = diagram
	s.diagramLock.Unlock()

	if s.notifyChan != nil {
		s.notifyChan <- struct{}{}
	}
}

// Start starts the web server.
func (s *WebServer) Start(ctx context.Context, addr string) error {
	if s.notifyChan == nil {
		s.notifyChan = make(chan struct{}, 1)
	}

	if s.Server == nil {
		s.Server = &http.Server{} //#nosec G112 - server is not exposed and the timeouts are default below
	}
	if s.Server.ReadTimeout == 0 {
		s.Server.ReadTimeout = readTimeout
	}
	if s.Server.ReadHeaderTimeout == 0 {
		s.Server.ReadHeaderTimeout = readTimeout
	}

	s.Server.Addr = addr
	s.Server.Handler = s.buildMux()

	// TODO: if the server fails the whole program should exit, could use RegisterOnShutdown for that
	go func() {
		if err := s.Server.ListenAndServe(); err != nil {
			s.Logger.Error(err, "web server stopped unexpectedly")
		}
	}()

	go func() {
		<-ctx.Done()
		s.Logger.Info("shutting down web server")

		timeoutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := s.Server.Shutdown(timeoutCtx); err != nil { //nolint:contextcheck
			s.Logger.Error(err, "failed to shutdown web server gracefully")
		}
	}()

	return nil
}
