// test3
//
// The fake client records every call it receives. Actions() returns them in
// order, and each action knows its verb and resource (Matches) and — for
// writes — carries the object that was sent. This is how you assert that a
// controller did the right WRITE, not just that it did not error.
//
// Inspect the recorded action to prove what was created.
//
// I AM NOT DONE
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx := context.Background()

	cs := fake.NewClientset()

	_, err := cs.CoreV1().ConfigMaps("demo").Create(ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "audited", Namespace: "demo"},
			Data:       map[string]string{"k": "v"},
		}, metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating configmap: %v", err)
	}

	actions := cs.Actions()
	exkit.AssertEqual("actions the fake recorded", len(actions), 1)

	// Matches checks (verb, resource). The call above was a create — but this
	// asserts "update", so it is false and the assertion fails. Name the verb
	// you actually called.
	exkit.AssertTrue("the recorded action was a create on configmaps", actions[0].Matches("update", "configmaps"))

	// A create action carries the object that was sent.
	create := actions[0].(k8stesting.CreateAction)
	obj := create.GetObject().(*corev1.ConfigMap)
	exkit.AssertEqual("the audited configmap's data", obj.Data["k"], "v")

	exkit.Successf("Actions() is your audit log — assert the write happened, with the payload you expected")
}
