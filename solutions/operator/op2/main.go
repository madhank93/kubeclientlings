// op2
//
// Owns() is how a controller notices its children. It maps an event on a
// child object back to a reconcile request for the OWNER — by reading the
// child's controller ownerRef. No ownerRef, no mapping: delete the child and
// nobody hears it. SetControllerReference is what stamps that ref.
//
// Stamp the parent's controller reference on the child before creating it.
package main

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// childMinder makes sure the ConfigMap named "parent" always has its child
// Secret. The child is stamped with a controller ownerRef so Owns() can map
// child events back to the parent.
type childMinder struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *childMinder) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var parent corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &parent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Every namespace also holds a kube-root-ca.crt ConfigMap; this
	// controller only manages the one named "parent".
	if parent.Name != "parent" {
		return ctrl.Result{}, nil
	}

	child := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "parent-child", Namespace: parent.Namespace},
		StringData: map[string]string{"owner": parent.Name},
	}
	// The ownerRef is the whole story: it is what Owns(&corev1.Secret{})
	// reads to turn a child event into a request for the parent.
	if err := controllerutil.SetControllerReference(&parent, child, r.scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, child); err != nil && !apierrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op2")
	defer cancel()

	ctrl.SetLogger(logr.Discard()) // keep exercise output clean

	mgr, err := ctrl.NewManager(exkit.MustRESTConfig(), ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{ns: {}}},
	})
	if err != nil {
		exkit.Failf("building manager: %v", err)
	}

	r := &childMinder{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r); err != nil {
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
		ObjectMeta: metav1.ObjectMeta{Name: "parent", Namespace: ns},
	}, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating parent configmap: %v", err)
	}

	exkit.WaitFor(ctx, "the controller to create the child secret", func(ctx context.Context) (bool, error) {
		_, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return err == nil, err
	})

	child, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("fetching child secret: %v", err)
	}
	ref := metav1.GetControllerOf(child)
	exkit.AssertTrue("child carries a controller ownerRef pointing at parent",
		ref != nil && ref.Kind == "ConfigMap" && ref.Name == "parent")

	// The payoff: kill the child. Its delete event carries the ownerRef,
	// Owns maps it to "parent", and the reconciler puts the child back.
	oldUID := child.UID
	if err := cs.CoreV1().Secrets(ns).Delete(ctx, "parent-child", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting child secret: %v", err)
	}
	exkit.WaitFor(ctx, "the controller to recreate the deleted child", func(ctx context.Context) (bool, error) {
		s, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return s.UID != oldUID, nil
	})

	exkit.Successf("Owns + ownerRefs: delete a child and the controller puts it right back")
}
