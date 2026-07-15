// crd1
//
// The dynamic client hands you unstructured maps, but your code wants a real
// Go struct. runtime.DefaultUnstructuredConverter bridges the two — and it
// maps fields by their `json` tags, exactly like encoding/json. A tag that
// does not match the key in the unstructured object leaves that field zero.
//
// Read a custom resource's spec into a typed struct.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/madhank93/clientlings/internal/exkit"
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
	// What the dynamic client would return for a Widget.
	u := map[string]any{
		"apiVersion": "clientlings.dev/v1alpha1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "w1"},
		"spec":       map[string]any{"size": int64(7)},
	}

	var w Widget
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &w); err != nil {
		exkit.Failf("converting unstructured to Widget: %v", err)
	}

	exkit.AssertEqual("the Kind the converter carried across", w.Kind, "Widget")
	exkit.AssertEqual("spec.size read into the typed field", w.Spec.Size, int64(7))

	exkit.Successf("DefaultUnstructuredConverter maps by json tag — typed structs over the dynamic client")
}
