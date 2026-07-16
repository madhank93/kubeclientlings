// pods5
//
// Delete() only STARTS a deletion — the object lingers while finalizers and
// the kubelet do their work. Code that needs the object gone must poll until
// Get returns a NotFound error. And NotFound here is not a failure: it is
// exactly the signal you are waiting for. apierrors.IsNotFound() is how you
// tell it apart from real errors.
//
// Delete the pod and wait until it is truly gone.
//
// I AM NOT DONE
package main

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods5")
	defer cancel()

	if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "doomed"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	if err := cs.CoreV1().Pods(ns).Delete(ctx, "doomed", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting pod: %v", err)
	}

	exkit.WaitFor(ctx, "pod to disappear", func(ctx context.Context) (bool, error) {
		_, err := cs.CoreV1().Pods(ns).Get(ctx, "doomed", metav1.GetOptions{})
		// This treats EVERY error as fatal — including the NotFound that
		// means the pod is finally gone. One kind of error here is not a
		// failure at all; check for it first (apierrors.IsNotFound).
		if err != nil {
			return false, err
		}
		return false, nil // still terminating
	})

	exkit.Successf("pod deleted and confirmed gone — NotFound was the success signal")
}
