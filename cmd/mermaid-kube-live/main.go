package main

import (
	"log"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Serve Serve `cmd:"" help:"Serve the diagram over HTTP."`
}

func main() {
	kctx := kong.Parse(&CLI)
	if err := kctx.Run(); err != nil {
		log.Fatal(err)
	}
}
