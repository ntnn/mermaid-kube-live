package main

type Generate struct {
	CommonFlags `embed:""`

	Scopes     []string `sep:"," short:"s" long:"scopes" help:"Comma-separated list of scopes to export" default:"cluster,namespaced"`
	Namespaces []string `sep:"," short:"n" long:"namespaces" help:"Comma-separated list of namespaces to export, all are exported if empty" default:""`
	// ExportCRDS bool     `help:"Print CRDs as nodes" default:"false"`
}

func (g *Generate) Run() error {
	return nil
}
