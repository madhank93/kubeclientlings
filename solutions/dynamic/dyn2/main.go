// dyn2
//
// Unstructured objects are map[string]any all the way down. Digging fields
// out by hand means type assertions at every level — the unstructured
// package's Nested* helpers do it safely: give the FULL path, get back
// (value, found, err). found=false is not an error; it just means nothing
// lives at that path.
//
// Read the ConfigMap value through the correct nested path.
package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("dyn2")
	defer cancel()

	_, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "greeting", Namespace: ns},
		Data:       map[string]string{"message": "hello from unstructured land"},
	}, metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating configmap: %v", err)
	}

	dyn := exkit.MustDynamic()
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	u, err := dyn.Resource(gvr).Namespace(ns).Get(ctx, "greeting", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting configmap: %v", err)
	}

	// The value lives at .data.message — the path is every key from the
	// top of the object down.
	msg, found, err := unstructured.NestedString(u.Object, "data", "message")
	if err != nil {
		exkit.Failf("reading nested field: %v", err)
	}

	exkit.AssertTrue("field found at path data.message", found)
	exkit.AssertEqual("the value", msg, "hello from unstructured land")
	exkit.Successf("Nested helpers walk the map for you — full path in, (value, found, err) out")
}
