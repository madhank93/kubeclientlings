// watch4
//
// Events — the things `kubectl describe` shows under "Events:" — don't
// appear by magic. A controller wires up an EventBroadcaster, points it at
// the API server with StartRecordingToSink, and gets an EventRecorder to
// emit through. Skip the sink and every event vanishes into the void:
// no errors, nothing stored.
//
// Wire the broadcaster to the API server so the event actually lands.
package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("watch4")
	defer cancel()

	pod, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "noisy"), metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	broadcaster := record.NewBroadcaster()
	// This is the missing link in the broken version: without a sink the
	// recorder emits into memory and nothing ever reaches the cluster.
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: cs.CoreV1().Events(ns),
	})
	defer broadcaster.Shutdown()

	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "kubeclientlings"})
	recorder.Event(pod, corev1.EventTypeNormal, "Exercised", "kubeclientlings was here")

	exkit.WaitFor(ctx, "the event to appear in the API", func(ctx context.Context) (bool, error) {
		events, err := cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, ev := range events.Items {
			if ev.Reason == "Exercised" && ev.InvolvedObject.Name == "noisy" {
				return true, nil
			}
		}
		return false, nil
	})

	exkit.Successf("event recorded — check it with: kubectl -n %s get events", ns)
}
