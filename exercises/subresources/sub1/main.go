// sub1
//
// Replicas live behind the /scale subresource, not the main object. That is
// why `kubectl scale` and an HPA can resize a Deployment without touching its
// spec: they GET the Scale, set Scale.Spec.Replicas — the DESIRED count — and
// UpdateScale. Scale.Status.Replicas is the observed count and is read-only on
// write; setting it does nothing.
//
// Resize a Deployment through its scale subresource.
//
// I AM NOT DONE
package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("sub1")
	defer cancel()

	one := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
			},
		},
	}
	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating the deployment: %v", err)
	}

	scale, err := cs.AppsV1().Deployments(ns).GetScale(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading the scale subresource: %v", err)
	}

	// Status.Replicas is the OBSERVED count and read-only on write — setting it
	// changes nothing, so the deployment stays at 1. The desired count is
	// Scale.Spec.Replicas; set that instead.
	scale.Status.Replicas = 3
	if _, err := cs.AppsV1().Deployments(ns).UpdateScale(ctx, "web", scale, metav1.UpdateOptions{}); err != nil {
		exkit.Failf("updating the scale subresource: %v", err)
	}

	got, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading the deployment back: %v", err)
	}
	exkit.AssertEqual("desired replicas after UpdateScale", *got.Spec.Replicas, int32(3))

	exkit.Successf("Scale.Spec.Replicas is the desired count — the /scale endpoint is how HPAs resize you")
}
