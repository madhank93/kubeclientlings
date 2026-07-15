// ssa2
//
// Two managers, one field: when a second manager applies a value to a field
// the first already owns, the apiserver returns a conflict — that is SSA
// protecting you from silent clobbering. To deliberately take ownership, you
// set Force. It resolves the conflict and moves the field into your name.
//
// Take over a field a different manager owns.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigcorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("ssa2")
	defer cancel()

	// mgr-a establishes ownership of data.owner.
	applyA := applyconfigcorev1.ConfigMap("shared", ns).WithData(map[string]string{"owner": "from-a"})
	if _, err := cs.CoreV1().ConfigMaps(ns).Apply(ctx, applyA, metav1.ApplyOptions{FieldManager: "mgr-a"}); err != nil {
		exkit.Failf("mgr-a applying: %v", err)
	}

	// mgr-b wants the same field. Without Force this conflicts; with Force it
	// steals ownership.
	applyB := applyconfigcorev1.ConfigMap("shared", ns).WithData(map[string]string{"owner": "from-b"})
	if _, err := cs.CoreV1().ConfigMaps(ns).Apply(ctx, applyB, metav1.ApplyOptions{FieldManager: "mgr-b", Force: true}); err != nil {
		exkit.Failf("mgr-b applying with force: %v", err)
	}

	got, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "shared", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading it back: %v", err)
	}
	exkit.AssertEqual("the field after a forced apply", got.Data["owner"], "from-b")

	exkit.Successf("conflicting managers are a feature — Force is how you knowingly take a field over")
}
