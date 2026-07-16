// dyn1
//
// The dynamic client works on ANY resource — no typed structs, just
// unstructured maps. The price: you must name the resource yourself with a
// GroupVersionResource, and the resource is the lowercase PLURAL from the
// URL path ("pods"), never the Kind ("Pod"). The server has no idea what
// /api/v1/namespaces/x/Pod is.
//
// List the pods through the dynamic client with the right GVR.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("dyn1")
	defer cancel()

	for _, name := range []string{"un-1", "un-2"} {
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, name), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", name, err)
		}
	}

	dyn := exkit.MustDynamic()

	// "Pod" is the KIND — the name that appears inside objects. URLs are
	// built from the RESOURCE, the lowercase plural. This GVR asks the
	// server for /api/v1/namespaces/<ns>/Pod, which doesn't exist.
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "Pod"}

	list, err := dyn.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		exkit.Failf("listing through the dynamic client: %v", err)
	}

	exkit.AssertEqual("pods seen by the dynamic client", len(list.Items), 2)
	exkit.AssertEqual("kind reported by the server", list.Items[0].GetKind(), "Pod")
	exkit.Successf("same pods, no typed structs — GVRs name resources, Kinds name objects")
}
