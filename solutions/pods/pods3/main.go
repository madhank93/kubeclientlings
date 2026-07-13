// pods3
//
// Update() is optimistic concurrency in action: the object you send must
// carry the resourceVersion of the object you read. You never build an
// update by hand — you Get the current object, mutate it, and send it back.
// And because ANYONE can write between your Get and your Update (the kubelet
// writes pod status constantly), production code wraps the cycle in
// retry.RetryOnConflict.
//
// Add the label tier=frontend to the existing pod, the right way.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods3")
	defer cancel()

	if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "hello"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	// Get the live object (it carries the resourceVersion), mutate, Update —
	// retrying if someone else (the kubelet!) wrote in between.
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
		if err != nil {
			return err
		}
		current.Labels["tier"] = "frontend"
		_, err = cs.CoreV1().Pods(ns).Update(ctx, current, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		exkit.Failf("updating pod: %v", err)
	}

	got, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading pod: %v", err)
	}
	exkit.AssertEqual("tier label after update", got.Labels["tier"], "frontend")
	exkit.Successf("read-modify-write cycle done — resourceVersion made the update safe")
}
