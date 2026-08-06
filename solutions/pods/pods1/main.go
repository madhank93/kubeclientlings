// pods1
//
// The most fundamental client-go operation: create an object, read it back.
// Namespaced clients take the namespace as an ARGUMENT — Pods(ns) — and that
// argument, not the object's metadata, decides which URL the request hits.
//
// Create the pod in the exercise namespace, then Get it back.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods1")
	defer cancel()

	pod := exkit.NginxPod(ns, "hello")

	// CoreV1() picks the group/version, Pods(ns) the resource and namespace:
	// together they build POST /api/v1/namespaces/<ns>/pods.
	_, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	// Reading it back returns the SERVER's copy — uid, resourceVersion and
	// every defaulted spec field filled in, unlike the object we sent.
	got, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting pod back: %v", err)
	}

	exkit.AssertEqual("pod name", got.Name, "hello")
	exkit.AssertEqual("pod namespace", got.Namespace, ns)
	exkit.Successf("created and fetched %s/%s — the CRUD basics work", ns, got.Name)
}
