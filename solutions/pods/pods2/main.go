// pods2
//
// List() returns everything in the namespace unless you narrow it down.
// The server does the filtering — you express it in ListOptions with a
// label selector string, exactly the same syntax as `kubectl get -l`.
//
// List only the pods labeled app=web.
package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("pods2")
	defer cancel()

	// Three web pods, two db pods.
	for i := 1; i <= 3; i++ {
		pod := exkit.NginxPod(ns, fmt.Sprintf("web-%d", i))
		pod.Labels = map[string]string{"app": "web"}
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", pod.Name, err)
		}
	}
	for i := 1; i <= 2; i++ {
		pod := exkit.NginxPod(ns, fmt.Sprintf("db-%d", i))
		pod.Labels = map[string]string{"app": "db"}
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", pod.Name, err)
		}
	}

	webPods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: "app=web",
	})
	if err != nil {
		exkit.Failf("listing pods: %v", err)
	}

	exkit.AssertEqual("pods matching app=web", len(webPods.Items), 3)
	exkit.Successf("server-side label filtering works — only the web pods came back")
}
