// mermaid-kube-live is a tool to colour mermaid diagrams based on live
// Kubernetes cluster data.
package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
)

type CommonFlags struct {
	Config     string   `default:"mkl.yaml"    help:"Configuration file" required:"true"                            short:"c"`
	Diagram    string   `default:"mkl.mermaid" help:"Diagram file"       required:"true"                            short:"d"`
	Kubeconfig []string `default:""            env:"KUBECONFIG"          help:"Comma-separated list of kubeconfigs" sep:","   short:"k"`
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
	Serve Serve `cmd:"" help:"Serve the diagram over HTTP."`
}

func main() {
	kctx := kong.Parse(&CLI)
	if err := kctx.Run(); err != nil {
		log.Fatal(err)
	}
}
