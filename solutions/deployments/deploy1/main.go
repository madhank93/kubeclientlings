// deploy1
//
// A Deployment's selector must match its pod template's labels — that is the
// contract that ties the Deployment to the pods it owns. The API server
// enforces it: a mismatch is rejected with a 422 "selector does not match
// template labels" before anything is created.
//
// Fix the labels so the Deployment is accepted.
package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("deploy1")
	defer cancel()

	replicas := int32(2)
	labels := map[string]string{"app": "web"}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "web", Image: exkit.Image}},
				},
			},
		},
	}

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	got, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("getting deployment: %v", err)
	}
	exkit.AssertEqual("selector matches template labels",
		got.Spec.Selector.MatchLabels["app"], got.Spec.Template.Labels["app"])
	exkit.Successf("deployment accepted — selector and template labels agree")
}
