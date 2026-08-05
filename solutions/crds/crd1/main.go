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

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

type Widget struct {
	// Inline: apiVersion/kind land on the OUTER object, not nested.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WidgetSpec `json:"spec"`
}

type WidgetSpec struct {
	// The tag must equal the map key exactly. A mismatch is not an error —
	// the field just stays zero, like a JSON document missing that key.
	// int64 because JSON has a single numeric type.
	Size int64 `json:"size"`
}

func main() {
	// What the dynamic client would return for a Widget.
	u := map[string]any{
		"apiVersion": "kubeclientlings.dev/v1alpha1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "w1"},
		"spec":       map[string]any{"size": int64(7)},
	}

	// Reflection over the map, matched by json tag — the same rules as
	// encoding/json, without the marshal round-trip. This is the seam that
	// lets typed code sit on top of the dynamic client.
	var w Widget
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &w); err != nil {
		exkit.Failf("converting unstructured to Widget: %v", err)
	}

	exkit.AssertEqual("the Kind the converter carried across", w.Kind, "Widget")
	exkit.AssertEqual("spec.size read into the typed field", w.Spec.Size, int64(7))

	exkit.Successf("DefaultUnstructuredConverter maps by json tag — typed structs over the dynamic client")
}
