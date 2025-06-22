package mkl

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ntnn/mermaid-kube-live/pkg/fileprovider"
)

var (
	provider    *fileprovider.Provider
	clusterName string
)

func namespace(t *testing.T) string {
	t.Helper()

	name := strings.ToLower(t.Name())

	namespaceCluster(t, name)

	return name
}

func getOrCreate(t *testing.T, client client.Client, obj client.Object) {
	t.Helper()

	objectKey := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}

	if err := client.Get(t.Context(), objectKey, obj); err == nil {
		require.NoError(t, client.Update(t.Context(), obj))
		return
	}
	require.NoError(t, client.Create(t.Context(), obj))
}

func namespaceCluster(t *testing.T, namespace string) {
	t.Helper()

	cluster, err := provider.Get(t.Context(), clusterName)
	require.NoError(t, err)

	client := cluster.GetClient()

	getOrCreate(t, client, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})

	t.Cleanup(func() {
		require.NoError(t, client.Delete(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}))
	})
}

func suite(ctx context.Context) error {
	kubeEnv := os.Getenv("KUBECONFIG")
	if kubeEnv == "" {
		kubeEnv = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	clusterName = os.Getenv("KUBECONTEXT")

	fmt.Fprintf(os.Stderr, "Integration tests are run against a live kubernetes cluster.\n")
	fmt.Fprintf(os.Stderr, "KUBECONFIG: %q\n", kubeEnv)
	fmt.Fprintf(os.Stderr, "KUBECONTEXT: %q (defaults to first cluster in kubeconfig)\n", clusterName)
	fmt.Fprintf(os.Stderr, "The tests will create resources in the cluster, and clean them up afterwards.\n")

	var err error
	provider, err = fileprovider.FromFiles(kubeEnv)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %w", err)
	}
	if err := provider.RunOnce(ctx); err != nil {
		return fmt.Errorf("failed to run provider: %w", err)
	}

	if _, err := provider.Get(context.Background(), clusterName); err != nil {
		return fmt.Errorf("failed to get cluster %q: %w", clusterName, err)
	}

	return nil
}

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := suite(ctx); err != nil {
			panic("Suite initialization failed: " + err.Error())
		}
	}
	m.Run()
}
