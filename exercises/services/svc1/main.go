// svc1
//
// A Service finds its backends with a label selector. Nothing validates it
// at create time — a Service with a selector that matches no pods is
// perfectly legal and silently routes to nowhere. This is one of the most
// common "why is my service not working" bugs in real clusters.
//
// Fix the Service's selector so it actually selects the web pods.
//
// I AM NOT DONE
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("svc1")
	defer cancel()

	// Two web pods, labeled app=web (the fixture labels them app=<name>).
	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 2), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns},
		Spec: corev1.ServiceSpec{
			// The pods are labeled app=web. This selector hunts for
			// app=webserver. Nobody will ever answer.
			Selector: map[string]string{"app": "webserver"},
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt32(80),
			}},
		},
	}
	if _, err := cs.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating service: %v", err)
	}

	// The deployment controller creates pods asynchronously — give them a
	// moment to exist before judging the service's selector.
	exkit.WaitFor(ctx, "the deployment's 2 pods to exist", func(ctx context.Context) (bool, error) {
		pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=web"})
		if err != nil {
			return false, err
		}
		return len(pods.Items) == 2, nil
	})

	// The moment of truth: does the service's selector select anything?
	selected, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(svc.Spec.Selector).String(),
	})
	if err != nil {
		exkit.Failf("listing pods with the service's selector: %v", err)
	}
	if len(selected.Items) == 0 {
		exkit.Failf("the service selects ZERO pods — its selector %v matches no pod labels.\nCompare it with the labels on the deployment's pods.", svc.Spec.Selector)
	}

	exkit.AssertEqual("pods selected by the service", len(selected.Items), 2)
	exkit.Successf("service selector matches the pods — traffic has somewhere to go")
}
