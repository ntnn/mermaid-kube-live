package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
)

type CommonFlags struct {
	Config     string   `short:"c" help:"Configuration file" required:"true" default:"config.yaml"`
	Diagram    string   `short:"d" help:"Diagram file" required:"true" default:"diagram.mermaid"`
	Kubeconfig []string `sep:"," short:"k" help:"Comma-separated list of kubeconfigs" env:"KUBECONFIG" default:""`
}

func (c CommonFlags) kubeconfig() []string {
	if len(c.Kubeconfig) > 0 {
		return c.Kubeconfig
	}
	return []string{
		os.ExpandEnv("$HOME/.kube/config"),
	}
}

var CLI struct {
	Serve    Serve    `cmd:"" help:"Serve the diagram over HTTP."`
	Generate Generate `cmd:"" help:"Generate the diagram and config file."`
}

func main() {
	kctx := kong.Parse(&CLI)
	if err := kctx.Run(); err != nil {
		log.Fatal(err)
	}
}
