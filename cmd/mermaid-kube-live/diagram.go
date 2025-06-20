package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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
	return nil
}
