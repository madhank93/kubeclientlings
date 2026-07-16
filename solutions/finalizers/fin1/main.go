// fin1
//
// A finalizer is how an operator says "wait — I have cleanup to do before this
// object goes away." While a finalizer is present, a Delete does NOT remove the
// object: the apiserver just stamps it with a DeletionTimestamp and leaves it,
// terminating, until the finalizer is cleared.
//
// Protect an object from deletion with a finalizer.
package main

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const finalizer = "kubeclientlings.dev/protect"

func main() {
	ctx, cancel, cs, ns := exkit.Begin("fin1")
	defer cancel()

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:       "guarded",
		Namespace:  ns,
		Finalizers: []string{finalizer},
	}}
	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating the configmap: %v", err)
	}

	if err := cs.CoreV1().ConfigMaps(ns).Delete(ctx, "guarded", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting the configmap: %v", err)
	}

	got, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "guarded", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		exkit.Failf("the object was removed — a finalizer should have blocked the delete")
	}
	if err != nil {
		exkit.Failf("reading it back: %v", err)
	}
	exkit.AssertTrue("delete only marked it terminating, not gone", got.DeletionTimestamp != nil)

	// Release it so the namespace can be torn down next run.
	got.Finalizers = nil
	if _, err := cs.CoreV1().ConfigMaps(ns).Update(ctx, got, metav1.UpdateOptions{}); err != nil {
		exkit.Failf("clearing the finalizer: %v", err)
	}

	exkit.Successf("a finalizer turns Delete into 'mark terminating' — the object waits for cleanup")
}
