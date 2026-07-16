// deploy2
//
// Scaling is just updating .spec.replicas — but the field is a *int32, a
// pointer, so "unset" can be told apart from "zero". Go won't let you take
// a pointer to an untyped constant or assign *int where *int32 is wanted;
// this trips up everyone exactly once.
//
// Scale the deployment to 3 replicas.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("deploy2")
	defer cancel()

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 1), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	dep, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting deployment: %v", err)
	}

	// This does not compile: replicas is an int, but the field wants *int32.
	// Look at the field's type and fix the declaration.
	replicas := 3
	dep.Spec.Replicas = &replicas

	if _, err := cs.AppsV1().Deployments(ns).Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
		exkit.Failf("scaling deployment: %v", err)
	}

	got, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading deployment: %v", err)
	}
	exkit.AssertEqual("desired replicas", *got.Spec.Replicas, int32(3))
	exkit.Successf("scaled to 3 — and now you know why replicas is *int32")
}
