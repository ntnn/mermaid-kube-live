package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	readTimeout     = 5 * time.Minute
	shutdownTimeout = 5 * time.Second
)

func (s *Serve) httpServer(ctx context.Context) {
	mux := http.NewServeMux()

	// Serve the main page
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(mainPage)); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})

	// Serve the built diagram
	mux.HandleFunc("/diagram", func(w http.ResponseWriter, _ *http.Request) {
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
	s.notifyChan = make(chan struct{})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
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

	server := &http.Server{
		Addr:        fmt.Sprintf("%s:%d", s.Host, s.Port),
		ReadTimeout: readTimeout,
		Handler:     mux,
	}

	log.Printf("starting webserver on %s", server.Addr)

	// TODO: if the server fails the whole program should exit.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("error from the web server: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("shutting down web server")

		timeoutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(timeoutCtx); err != nil { //nolint:contextcheck
			log.Printf("error shutting down web server: %v", err)
		}
	}()
}
