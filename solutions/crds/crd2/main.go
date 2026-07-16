// crd2
//
// Going the other way — typed struct to unstructured — is how you hand a Go
// value to the dynamic client. The catch: the dynamic client routes the
// request by apiVersion and kind, and those come from the embedded TypeMeta.
// Forget to set TypeMeta and the resulting object has no kind, so the client
// has no idea what endpoint to send it to.
//
// Convert a typed Widget into an unstructured object the dynamic client can send.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WidgetSpec `json:"spec"`
}

type WidgetSpec struct {
	Size int64 `json:"size"`
}

func main() {
	w := Widget{
		TypeMeta:   metav1.TypeMeta{APIVersion: "clientlings.dev/v1alpha1", Kind: "Widget"},
		ObjectMeta: metav1.ObjectMeta{Name: "w1"},
		Spec:       WidgetSpec{Size: 7},
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&w)
	if err != nil {
		exkit.Failf("converting Widget to unstructured: %v", err)
	}
	u := &unstructured.Unstructured{Object: m}

	exkit.AssertEqual("the kind the dynamic client will route on", u.GetKind(), "Widget")
	exkit.AssertEqual("the apiVersion the dynamic client will route on", u.GetAPIVersion(), "clientlings.dev/v1alpha1")

	size, found, err := unstructured.NestedInt64(m, "spec", "size")
	if err != nil || !found {
		exkit.Failf("reading spec.size back out: found=%v err=%v", found, err)
	}
	exkit.AssertEqual("spec.size survived the round trip", size, int64(7))

	exkit.Successf("ToUnstructured needs TypeMeta set — that is what the dynamic client routes on")
}
