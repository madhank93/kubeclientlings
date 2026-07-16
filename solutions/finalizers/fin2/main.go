// fin2
//
// The other half of the finalizer contract: once your cleanup is done, you
// remove your finalizer, and only then does the apiserver actually delete the
// object. A terminating object with a non-empty finalizer list lingers
// forever — clearing the list is what lets it go.
//
// Release a terminating object by removing its finalizer.
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const finalizer = "kubeclientlings.dev/protect"

func main() {
	ctx, cancel, cs, ns := exkit.Begin("fin2")
	defer cancel()

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:       "guarded",
		Namespace:  ns,
		Finalizers: []string{finalizer},
	}}
	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating the configmap: %v", err)
	}

	// Delete just marks it terminating — the finalizer holds it.
	if err := cs.CoreV1().ConfigMaps(ns).Delete(ctx, "guarded", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting the configmap: %v", err)
	}

	// Cleanup is done — clear the finalizer so the apiserver can remove it.
	got, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "guarded", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading the terminating object: %v", err)
	}
	got.Finalizers = nil
	if _, err := cs.CoreV1().ConfigMaps(ns).Update(ctx, got, metav1.UpdateOptions{}); err != nil {
		exkit.Failf("removing the finalizer: %v", err)
	}

	exkit.WaitFor(ctx, "the object to actually disappear once unfinalized", func(ctx context.Context) (bool, error) {
		_, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "guarded", metav1.GetOptions{})
		return apierrors.IsNotFound(err), nil
	})

	exkit.Successf("clearing the last finalizer is what finally lets the apiserver delete the object")
}
