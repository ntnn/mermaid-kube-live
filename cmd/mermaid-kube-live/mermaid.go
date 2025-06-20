package main

import (
	_ "embed"
	"log"
	"net/http"
)

var (
	//go:embed mermaid.html
	mainPage string
)

func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(mainPage)); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	})
}
