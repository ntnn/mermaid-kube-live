package main

type Generate struct {
	ConfigFile      string `short:"c" long:"config" help:"Path to write config to" default:"config.yaml" required:"true"`
	DiagramFile     string `short:"d" long:"diagram" help:"Path to write the diagram to" default:"diagram.mermaid" required:"true"`
	KubeconfigFiles string `short:"k" long:"kubeconfig-files" help:"Comma-separated list of kubeconfig files to use for clusters" required:"true"`
}

func (g *Generate) Run() error {
	return nil
}
