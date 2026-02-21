// mermaid-kube-live is a tool to colour mermaid diagrams based on live Kubernetes cluster data.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ntnn/mermaid-kube-live/pkg/mkl"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	mctrl "sigs.k8s.io/multicluster-runtime"
	"sigs.k8s.io/multicluster-runtime/providers/file"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	opts := &mkl.Options{}
	fs := opts.FlagSet()

	fDebug := fs.Bool("debug", false, "Enable debug logging")
	fKubeconfig := fs.String("kubeconfig", "", "Comma-separated list of kubeconfigs (default: $HOME/.kube/config)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Not pretty but the klog flags are a bit much.
	if *fDebug {
		klogFs := flag.NewFlagSet("klog", flag.ExitOnError)
		klog.InitFlags(klogFs)

		if err := klogFs.Set("v", "6"); err != nil {
			return fmt.Errorf("error setting klog verbosity: %w", err)
		}
	}

	logger := klog.Background()
	ctrl.SetLogger(logger)
	ctx := klog.NewContext(mctrl.SetupSignalHandler(), logger)
	opts.Logger = logger

	provider, err := file.New(file.Options{
		KubeconfigFiles: parseKubeconfigPaths(*fKubeconfig),
	})
	if err != nil {
		return fmt.Errorf("error setting up provider: %w", err)
	}
	opts.Provider = provider

	instance, err := mkl.New(opts)
	if err != nil {
		return fmt.Errorf("error creating MKL: %w", err)
	}

	return instance.Run(ctx)
}

func parseKubeconfigPaths(kubeconfigs string) []string {
	if kubeconfigs != "" {
		return strings.Split(kubeconfigs, ",")
	}

	return []string{
		os.ExpandEnv("$HOME/.kube/config"),
	}
}
