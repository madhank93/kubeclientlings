// op1
//
// Your first controller-runtime Reconciler. The manager owns all the
// machinery you built by hand in ctrl2 (informer → queue → worker); you only
// write Reconcile. The contract: a Request names one object, you return a
// Result plus error. NotFound is NORMAL — the object was deleted and the
// cache already forgot it. Returning an error there makes the manager retry
// a reconcile that can never succeed.
//
// Fetch the object the request names, and treat NotFound as "done".
//
// I AM NOT DONE
package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// labeler stamps reconciled=true on every ConfigMap it sees, and counts what
// happened so the exercise can assert on it.
type labeler struct {
	client.Client
	seen atomic.Int64 // reconciles entered
	errs atomic.Int64 // unexpected failures
}

func (r *labeler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.seen.Add(1)

	// A Request already carries the object's full namespace/name — but this
	// key is built by hand and loses the namespace, and EVERY error (even
	// "it's gone") is treated as a failure to retry forever.
	var cm corev1.ConfigMap
	key := types.NamespacedName{Name: req.Name}
	if err := r.Get(ctx, key, &cm); err != nil {
		r.errs.Add(1)
		return ctrl.Result{}, err
	}
	if cm.Labels["reconciled"] == "true" {
		return ctrl.Result{}, nil
	}

	// MergeFrom-patch instead of Update: idempotent and immune to the
	// stale-cache conflict a second event would otherwise cause.
	patch := client.MergeFrom(cm.DeepCopy())
	if cm.Labels == nil {
		cm.Labels = map[string]string{}
	}
	cm.Labels["reconciled"] = "true"
	if err := r.Patch(ctx, &cm, patch); err != nil {
		r.errs.Add(1)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op1")
	defer cancel()

	ctrl.SetLogger(logr.Discard()) // keep exercise output clean

	mgr, err := ctrl.NewManager(exkit.MustRESTConfig(), ctrl.Options{
		Scheme:  scheme.Scheme,                           // client-go's built-in types
		Metrics: metricsserver.Options{BindAddress: "0"}, // no metrics port
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{ns: {}}},
	})
	if err != nil {
		exkit.Failf("building manager: %v", err)
	}

	r := &labeler{Client: mgr.GetClient()}
	if err := ctrl.NewControllerManagedBy(mgr).For(&corev1.ConfigMap{}).Complete(r); err != nil {
		exkit.Failf("wiring controller: %v", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			exkit.Failf("manager exited: %v", err)
		}
	}()
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		exkit.Failf("manager cache never synced")
	}

	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: ns},
	}, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating configmap: %v", err)
	}

	exkit.WaitFor(ctx, "the reconciler to label app-config", func(ctx context.Context) (bool, error) {
		if n := r.errs.Load(); n > 0 {
			return false, fmt.Errorf("the reconciler reported %d error(s) — check how Get and its error are handled", n)
		}
		cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "app-config", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return cm.Labels["reconciled"] == "true", nil
	})

	// Now the delete: the reconciler gets one more request for an object
	// that no longer exists. A correct Reconcile treats that as success.
	before := r.seen.Load()
	if err := cs.CoreV1().ConfigMaps(ns).Delete(ctx, "app-config", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting configmap: %v", err)
	}
	exkit.WaitFor(ctx, "the delete event to reach the reconciler", func(ctx context.Context) (bool, error) {
		return r.seen.Load() > before, nil
	})
	exkit.AssertEqual("reconcile errors after the delete", r.errs.Load(), int64(0))

	exkit.Successf("Request in, Result out, NotFound is normal: that is the whole Reconcile contract")
}
