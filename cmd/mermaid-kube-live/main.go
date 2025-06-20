package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	flag.Parse()
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

var (
	fHost = flag.String("host", "localhost", "Host to listen on")
	fPort = flag.Int("port", 8080, "Port to listen on")
)

func doMain() error {
	if err := loadDiagram(); err != nil {
		return fmt.Errorf("failed to load diagram: %w", err)
	}

	notifyChan := make(chan struct{})

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		defer w.(http.CloseNotifier).CloseNotify()
		for range notifyChan {
			fmt.Fprintf(w, "data: diagram updated\n\n")
			w.(http.Flusher).Flush()
		}
	})

	// TODO poll node data from kube clusters

	// TODO add colouring based on node status

	// TODO update diagram and notify clients

	go func() {
		for range time.Tick(1 * time.Second) {
			log.Println("Simulating diagram update...")
			// Simulate a diagram update
			notifyChan <- struct{}{}
		}
	}()

	return http.ListenAndServe(fmt.Sprintf("%s:%d", *fHost, *fPort), nil)
}
