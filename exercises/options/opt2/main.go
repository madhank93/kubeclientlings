// opt2
//
// Field selectors filter on OBJECT FIELDS instead of labels — but only on
// the handful of fields the API server indexes (metadata.name,
// metadata.namespace, status.phase, spec.nodeName, ...). Ask for anything
// else and the server rejects the whole request.
//
// Fetch just the pod named web-2 using a field selector.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("opt2")
	defer cancel()

	for _, name := range []string{"web-1", "web-2", "web-3"} {
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, name), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", name, err)
		}
	}

	// metadata.name is not a LABEL — no pod carries it as one, so this
	// label selector matches nothing (and that's not even an error, just an
	// empty list). The name lives in an object FIELD; pick the right kind
	// of selector.
	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: "metadata.name=web-2",
	})
	if err != nil {
		exkit.Failf("listing with field selector: %v", err)
	}

	exkit.AssertEqual("pods matching the field selector", len(pods.Items), 1)
	exkit.AssertEqual("the matched pod", pods.Items[0].Name, "web-2")
	exkit.Successf("field selectors work — but only on the fields the server indexes")
}
