package mermaidkubelive

import (
	"github.com/ntnn/mermaid-kube-live/pkg/fileprovider"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

var provider multicluster.Provider

func init() {
	var err error
	provider, err = fileprovider.FromDirectory("../../")
	if err != nil {
		panic(err)
	}
}
