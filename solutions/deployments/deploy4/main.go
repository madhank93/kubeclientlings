// deploy4
//
// "Is the rollout done?" is subtler than it looks. Status.Replicas counts
// pods that EXIST, not pods that are READY — and the status you read might
// describe an older generation of the spec. A correct rollout wait checks:
//   - ObservedGeneration >= Generation  (status describes MY change)
//   - UpdatedReplicas == desired        (all pods are on the new template)
//   - ReadyReplicas   == desired        (and they all pass readiness)
//
// Wait for the rollout properly. The pods take ~5s to become ready
// (readiness probe delay), which is exactly the window where sloppy
// checks declare victory too early.
package main

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("deploy4")
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
					Containers: []corev1.Container{{
						Name:  "web",
						Image: exkit.Image,
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: 5,
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/", Port: intstr.FromInt32(80)},
							},
						},
					}},
				},
			},
		},
	}

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	exkit.WaitFor(ctx, "rollout to complete (updated AND ready)", func(ctx context.Context) (bool, error) {
		d, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if d.Status.ObservedGeneration < d.Generation {
			return false, nil // status is stale — describes an older spec
		}
		return d.Status.UpdatedReplicas == replicas && d.Status.ReadyReplicas == replicas, nil
	})

	got, err := cs.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("re-reading deployment: %v", err)
	}
	exkit.AssertEqual("ready replicas", got.Status.ReadyReplicas, int32(2))
	exkit.Successf("rollout confirmed done — generation observed, all replicas updated and ready")
}
