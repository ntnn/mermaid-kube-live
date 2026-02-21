package styler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"slices"

	"github.com/go-logr/logr"
	"github.com/ntnn/mcutils"
	mklv1alpha1 "github.com/ntnn/mermaid-kube-live/apis/v1alpha1"
	"github.com/ntnn/mermaid-kube-live/pkg/multiplexer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mctrl "sigs.k8s.io/multicluster-runtime"
	mccontroller "sigs.k8s.io/multicluster-runtime/pkg/controller"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
)

type nodeHash string

func hashNode(nodeName string, node mklv1alpha1.Node) nodeHash {
	h := fnv.New64a()

	_, _ = h.Write([]byte(nodeName))
	if nodeBytes, err := json.Marshal(node); err == nil {
		_, _ = h.Write(nodeBytes)
	}

	return nodeHash(hex.EncodeToString(h.Sum(nil)))
}

type watches struct {
	logger         logr.Logger
	mp             *multiplexer.Multiplexer
	reconcilerOpts reconcilerOpts
	// cancel functions to track the watches for each node so they can be
	// stopped when the node is removed from the config
	cancels map[nodeHash]context.CancelFunc
}

func newWatches(mp *multiplexer.Multiplexer, reconcilerOpts reconcilerOpts) *watches {
	return &watches{
		logger:         mctrl.Log.WithName("watches"),
		mp:             mp,
		reconcilerOpts: reconcilerOpts,
		cancels:        make(map[nodeHash]context.CancelFunc),
	}
}

func (w *watches) update(ctx context.Context, nodes map[string]mklv1alpha1.Node) error {
	w.logger.V(2).Info("updating watches")
	hashed := map[nodeHash]mklv1alpha1.Node{}
	for nodeName, node := range nodes {
		hashed[hashNode(nodeName, node)] = node
	}

	// stop watches for any nodes that are no longer in the config
	keys := slices.Collect(maps.Keys(hashed))
	for hash, cancel := range w.cancels {
		if !slices.Contains(keys, hash) {
			w.logger.V(2).Info("stopping watch for node hash", "nodeHash", hash)
			cancel()
			delete(w.cancels, hash)
			w.mp.DeleteAware(string(hash))
		}
	}

	// start watches for any new / updated nodes
	var errs error

	for nodeName, node := range nodes {
		hash := hashNode(nodeName, node)
		if _, ok := w.cancels[hash]; ok {
			// node is already being watched
			continue
		}

		ctx, cancel := context.WithCancel(ctx)

		w.logger.V(2).Info("starting watch for node", "nodeName", nodeName, "nodeHash", hash)
		c, err := w.startUnmanaged(ctx, nodeName, node)
		if err != nil {
			w.logger.Error(err, "failed to start watch for node", "nodeName", nodeName)
			errs = fmt.Errorf("%w; failed to start watch for node %s: %w", errs, nodeName, err)
			cancel()
			continue
		}

		if err := w.mp.AddAware(ctx, string(hash), c); err != nil {
			w.logger.Error(err, "failed to add watch for node to multiplexer", "nodeName", nodeName)
			errs = fmt.Errorf("%w; failed to add watch for node %s to multiplexer: %w", errs, nodeName, err)
			cancel()
			continue
		}

		w.cancels[hash] = cancel
	}

	return errs
}

func (w *watches) startUnmanaged(ctx context.Context, nodeName string, node mklv1alpha1.Node) (mccontroller.Controller, error) { //nolint:cyclop
	logger := w.logger.WithValues("node", nodeName)
	logger.V(2).Info("creating unmanaged controller",
		"gvk", node.Selector.GVK.String(),
		"namespace", node.Selector.Namespace,
		"cluster", node.Selector.ClusterName,
		"owner", node.Selector.Owner,
	)

	// Build predicates based on the node's selector (cluster is handled later)
	predicates := []predicate.TypedPredicate[client.Object]{}

	if node.Selector.Name != "" {
		predicates = append(predicates, predicate.NewPredicateFuncs(func(obj client.Object) bool {
			if node.Selector.Namespace != "" {
				return obj.GetName() == node.Selector.Name && obj.GetNamespace() == node.Selector.Namespace
			}
			return obj.GetName() == node.Selector.Name
		}))
	}

	if node.Selector.LabelSelector.MatchLabels != nil || node.Selector.LabelSelector.MatchExpressions != nil {
		logger.V(2).Info("adding label selector predicate",
			"matchLabels", node.Selector.LabelSelector.MatchLabels,
			"matchExpressions", node.Selector.LabelSelector.MatchExpressions,
		)

		labelPredicate, err := predicate.LabelSelectorPredicate(node.Selector.LabelSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector: %w", err)
		}

		predicates = append(predicates, labelPredicate)
	}

	if node.Selector.Owner.Name != "" {
		logger.V(2).Info("adding owner predicate",
			"ownerGVK", node.Selector.Owner.GVK.String(),
			"ownerName", node.Selector.Owner.Name,
		)

		predicates = append(predicates, predicate.NewPredicateFuncs(func(obj client.Object) bool {
			ownerRef := metav1.GetControllerOf(obj)
			if ownerRef == nil {
				logger.V(3).Info("object has no controller owner", "object", obj.GetName())
				return false
			}

			if ownerRef.APIVersion == node.Selector.Owner.GVK.GroupVersion().String() &&
				ownerRef.Kind == node.Selector.Owner.GVK.Kind &&
				ownerRef.Name == node.Selector.Owner.Name {
				logger.V(3).Info("object matches owner predicate", "object", obj.GetName())
				return true
			}

			logger.V(3).Info("object owner does not match",
				"object", obj.GetName(),
				"ownerAPIVersion", ownerRef.APIVersion,
				"ownerKind", ownerRef.Kind,
				"ownerName", ownerRef.Name,
			)

			return false
		}))
	}

	// build source
	watchObj := &unstructured.Unstructured{}
	watchObj.SetGroupVersionKind(node.Selector.GVK)

	// Create an unmanaged reconciler so it can be stopped again
	r := reconciler{
		logger:   logger.WithValues("node", nodeName),
		opts:     w.reconcilerOpts,
		nodeName: nodeName,
		node:     node,
	}

	opts := mccontroller.Options{
		// We don't care about metrics.
		SkipNameValidation: ptr.To(true),
		Reconciler:         r,
	}

	c, err := mcutils.UnmanagedController(nodeName, watchObj, multicluster.ClusterName(node.Selector.ClusterName), predicates, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create unmanaged controller for node %s: %w", nodeName, err)
	}

	logger.Info("multi-cluster watch configured successfully")

	// start the controller
	go func() {
		logger.Info("starting controller goroutine")
		if err := c.Start(ctx); err != nil {
			logger.Error(err, "controller stopped with error")
		}
	}()

	logger.Info("unmanaged controller setup complete, waiting for watches to become active")

	return c, nil
}

type reconcilerOpts struct {
	getCluster      func(ctx context.Context, name multicluster.ClusterName) (cluster.Cluster, error)
	deleteResource  func(nodeName, name, namespace string)
	replaceResource func(nodeName string, resource unstructured.Unstructured)
	updateStyling   func(ctx context.Context, nodeName string, node mklv1alpha1.Node) error
}

type reconciler struct {
	logger   logr.Logger
	opts     reconcilerOpts
	nodeName string
	node     mklv1alpha1.Node
}

func (r reconciler) Reconcile(ctx context.Context, req mctrl.Request) (mctrl.Result, error) {
	logger := r.logger.WithValues("node", r.nodeName, "resource", req.NamespacedName.String(), "cluster", req.ClusterName)
	logger.Info("reconcile triggered")

	cl, err := r.opts.getCluster(ctx, req.ClusterName)
	if err != nil {
		return mctrl.Result{}, fmt.Errorf("failed to get cluster %s: %w", req.ClusterName, err)
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(r.node.Selector.GVK)

	if err := cl.GetClient().Get(ctx, req.NamespacedName, u); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to get resource")
			return mctrl.Result{}, fmt.Errorf("failed to get resource %s/%s in cluster %s: %w", req.Namespace, req.Name, req.ClusterName, err)
		}

		logger.Info("resource not found, deleting from tracking")
		r.opts.deleteResource(r.nodeName, req.Name, req.Namespace)

		return mctrl.Result{}, r.opts.updateStyling(ctx, r.nodeName, r.node)
	}

	logger.V(2).Info("resource found, updating", "labels", u.GetLabels())
	r.opts.replaceResource(r.nodeName, *u)

	return mctrl.Result{}, r.opts.updateStyling(ctx, r.nodeName, r.node)
}
