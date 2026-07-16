// ssa1
//
// Server-Side Apply lets you declare the state you want and let the apiserver
// merge it — but it tracks WHO owns each field, so every apply must name a
// field manager. Leave it empty and the apiserver rejects the request: it has
// nowhere to record ownership.
//
// Apply a ConfigMap with a field manager.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigcorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("ssa1")
	defer cancel()

	apply := applyconfigcorev1.ConfigMap("settings", ns).
		WithData(map[string]string{"replicas": "3"})

	_, err := cs.CoreV1().ConfigMaps(ns).Apply(ctx, apply, metav1.ApplyOptions{FieldManager: "clientlings"})
	if err != nil {
		exkit.Failf("applying the configmap: %v", err)
	}

	got, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "settings", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading it back: %v", err)
	}
	exkit.AssertEqual("the applied value", got.Data["replicas"], "3")

	exkit.Successf("SSA needs a FieldManager — that name is how the apiserver records who owns each field")
}
