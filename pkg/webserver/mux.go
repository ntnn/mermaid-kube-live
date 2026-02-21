package webserver

import (
	"fmt"
	"log"
	"net/http"

	_ "embed"
)

//go:embed serve.html
var mainPage string

func (s *WebServer) buildMux() *http.ServeMux {
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
		s.diagramLock.RLock()
		ret := make([]byte, len(s.diagram))
		copy(ret, s.diagram)
		s.diagramLock.RUnlock()

		if _, err := w.Write(ret); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})

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

	return mux
}
