// pods1
//
// The most fundamental client-go operation: create an object, read it back.
// Namespaced clients take the namespace as an ARGUMENT — Pods(ns) — and that
// argument, not the object's metadata, decides which URL the request hits.
//
// Create the pod in the exercise namespace, then Get it back.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods1")
	defer cancel()

	pod := exkit.NginxPod(ns, "hello")

	// The pod object says which namespace it wants... but the CLIENT was
	// given an empty one. Which of the two decides where the request goes?
	_, err := cs.CoreV1().Pods("").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	got, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting pod back: %v", err)
	}

	exkit.AssertEqual("pod name", got.Name, "hello")
	exkit.AssertEqual("pod namespace", got.Namespace, ns)
	exkit.Successf("created and fetched %s/%s — the CRUD basics work", ns, got.Name)
}
