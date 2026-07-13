// opt1
//
// Building selector strings by hand gets error-prone fast. The labels
// package builds them programmatically: labels.Set is a map, and
// SelectorFromSet turns it into a selector that requires EVERY key=value
// in the set (logical AND).
//
// Select exactly the prod web pods.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("opt1")
	defer cancel()

	fixtures := []struct {
		name string
		lbls map[string]string
	}{
		{"web-prod-1", map[string]string{"env": "prod", "tier": "web"}},
		{"web-prod-2", map[string]string{"env": "prod", "tier": "web"}},
		{"db-prod-1", map[string]string{"env": "prod", "tier": "db"}},
		{"web-dev-1", map[string]string{"env": "dev", "tier": "web"}},
	}
	for _, f := range fixtures {
		pod := exkit.NginxPod(ns, f.name)
		pod.Labels = f.lbls
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", f.name, err)
		}
	}

	selector := labels.SelectorFromSet(labels.Set{"env": "prod", "tier": "web"})

	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		exkit.Failf("listing pods: %v", err)
	}

	exkit.AssertEqual("prod web pods", len(pods.Items), 2)
	exkit.Successf("selector %q matched exactly the prod web pods", selector.String())
}
