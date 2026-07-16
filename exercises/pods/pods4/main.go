// pods4
//
// Patch() changes just the fields you send — no read-modify-write, no
// resourceVersion conflicts. A strategic merge patch is plain JSON shaped
// like the object itself, and the patch TYPE argument tells the server how
// to merge it.
//
// Patch the label tier=frontend onto the pod.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods4")
	defer cancel()

	if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "hello"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	// The patch body is shaped like the object — that's a merge-style patch.
	// But the type argument says JSONPatchType, and a JSON patch is a totally
	// different format (an array of op/path/value). Type and body must agree.
	patch := []byte(`{"metadata":{"labels":{"tier":"frontend"}}}`)
	_, err := cs.CoreV1().Pods(ns).Patch(ctx, "hello", types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		exkit.Failf("patching pod: %v", err)
	}

	got, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading pod: %v", err)
	}
	exkit.AssertEqual("tier label after patch", got.Labels["tier"], "frontend")
	exkit.AssertEqual("app label survived the merge", got.Labels["app"], "hello")
	exkit.Successf("strategic merge patch applied — existing labels merged, not replaced")
}
