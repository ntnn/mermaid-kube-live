package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/ntnn/mermaid-kube-live/pkg/fileprovider"
)

var (
	fKubeconfigDir   = flag.String("kubeconfig-dir", "", "Directory containing kubeconfig files for clusters")
	fKubeconfigFiles = flag.String("kubeconfig-files", "", "Comma-separated list of kubeconfig files to use for clusters")
)

func getProvider() (*fileprovider.Provider, error) {
	if *fKubeconfigDir != "" && *fKubeconfigFiles != "" {
		return nil, fmt.Errorf("cannot specify both --kubeconfig-dir and --kubeconfig-files")
	}

	if *fKubeconfigDir != "" {
		return fileprovider.FromDirectory(*fKubeconfigDir)
	}

	return fileprovider.FromFiles(strings.Split(*fKubeconfigFiles, ",")...)
}
