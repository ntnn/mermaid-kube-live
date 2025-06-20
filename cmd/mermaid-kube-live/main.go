package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	flag.Parse()
	if err := doMain(context.Background()); err != nil {
		log.Fatal(err)
	}
}

var (
	fHost = flag.String("host", "localhost", "Host to listen on")
	fPort = flag.Int("port", 8080, "Port to listen on")
)

// notifyChan is fed and closed by updateDiagramLoop.
var notifyChan = make(chan struct{}, 10)

func doMain(ctx context.Context) error {
	log.Printf("start diagram loop")
	if err := updateDiagramLoop(ctx); err != nil {
		return fmt.Errorf("failed to start diagram update loop: %w", err)
	}

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

	// go func() {
	// 	for range time.Tick(1 * time.Second) {
	// 		log.Println("Simulating diagram update...")
	// 		// Simulate a diagram update
	// 		notifyChan <- struct{}{}
	// 	}
	// }()

	log.Printf("listening on %s:%d", *fHost, *fPort)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", *fHost, *fPort), nil)
}
