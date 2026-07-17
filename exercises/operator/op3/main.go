// op3
//
// controllerutil.CreateOrUpdate is the reconciler's read-modify-write: it
// GETS the live object into your struct, runs your mutate() to lay desired
// state over it, then Creates or Updates only if something changed. Desired
// state set OUTSIDE mutate() is wiped by that Get — creation works once, but
// the update path becomes a no-op and drift is never repaired.
//
// Move the desired state (and the ownerRef) into the mutate function.
//
// I AM NOT DONE
package main

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// infoWriter mirrors each Deployment's replica count into a <name>-info
// ConfigMap, owned by the Deployment so drift on the child re-triggers it.
type infoWriter struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *infoWriter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      dep.Name + "-info",
		Namespace: dep.Namespace,
	}}
	// Desired state prepared up front… but CreateOrUpdate GETS the live
	// object into cm first, wiping these fields before the (empty) mutate fn
	// runs. Creation works once; the update path never updates anything.
	cm.Data = map[string]string{"replicas": strconv.Itoa(int(*dep.Spec.Replicas))}
	if err := controllerutil.SetControllerReference(&dep, cm, r.scheme); err != nil {
		return ctrl.Result{}, err
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		return nil
	})
	return ctrl.Result{}, err
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op3")
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

	r := &infoWriter{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
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

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 1), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	replicasIs := func(want string) func(context.Context) (bool, error) {
		return func(ctx context.Context) (bool, error) {
			cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "web-info", metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return cm.Data["replicas"] == want, nil
		}
	}

	// Create path.
	exkit.WaitFor(ctx, "web-info to record replicas=1", replicasIs("1"))

	// Update path: scale the deployment, the mirror must follow.
	if _, err := cs.AppsV1().Deployments(ns).Patch(ctx, "web",
		types.StrategicMergePatchType, []byte(`{"spec":{"replicas":3}}`), metav1.PatchOptions{}); err != nil {
		exkit.Failf("scaling deployment: %v", err)
	}
	exkit.WaitFor(ctx, "web-info to follow the scale-up (replicas=3)", replicasIs("3"))

	// Drift repair: tamper with the child, the ownerRef event brings the
	// reconciler back and CreateOrUpdate restores desired state.
	if _, err := cs.CoreV1().ConfigMaps(ns).Patch(ctx, "web-info",
		types.StrategicMergePatchType, []byte(`{"data":{"replicas":"999"}}`), metav1.PatchOptions{}); err != nil {
		exkit.Failf("tampering with web-info: %v", err)
	}
	exkit.WaitFor(ctx, "the controller to repair the drifted ConfigMap", replicasIs("3"))

	exkit.Successf("CreateOrUpdate with a real mutate fn: create, update, and drift repair from one code path")
}
