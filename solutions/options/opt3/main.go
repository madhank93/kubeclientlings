// opt3
//
// Server-side apply (SSA) is the modern way to manage objects: you declare
// the fields you care about, the server merges, and every field remembers
// its OWNER (the field manager). That's why ApplyOptions.FieldManager is
// mandatory — apply without an identity is meaningless.
//
// Apply the pod with a field manager so the server knows who owns what.
package main

import (
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("opt3")
	defer cancel()

	// Apply configurations are generated structs where EVERY field is a
	// pointer, so "not set" is representable — with a plain corev1.Pod the
	// zero values would read as a claim of ownership over those fields.
	pod := corev1apply.Pod("hello", ns).
		WithLabels(map[string]string{"managed-by": "ssa"}).
		WithSpec(corev1apply.PodSpec().
			WithContainers(corev1apply.Container().
				WithName("web").
				WithImage(exkit.Image)))

	// Required on Apply (unlike Update, where client-go defaults it): the
	// server records ownership per field under this name. Keep it stable —
	// a fresh name per run orphans the fields the old one owned.
	applied, err := cs.CoreV1().Pods(ns).Apply(ctx, pod, metav1.ApplyOptions{
		FieldManager: "kubeclientlings",
	})
	if err != nil {
		exkit.Failf("applying pod: %v", err)
	}

	exkit.AssertEqual("label set by apply", applied.Labels["managed-by"], "ssa")

	// metadata.managedFields is the server's per-field ownership ledger — one
	// entry per manager that has ever written to this object.
	manager := ""
	for _, mf := range applied.ManagedFields {
		if mf.Manager == "kubeclientlings" {
			manager = mf.Manager
		}
	}
	exkit.AssertEqual("field manager recorded by the server", manager, "kubeclientlings")
	exkit.Successf("server-side apply done — the server tracks 'kubeclientlings' as the owner of those fields")
}
