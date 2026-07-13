// watch2
//
// Where a watch STARTS matters. ResourceVersion "0" replays the server's
// cached state first — every existing object arrives as a fresh Added
// event. The list-then-watch pattern avoids that: List, remember the
// list's resourceVersion, watch from exactly there. Only genuinely new
// changes flow.
//
// Start the watch from the right resourceVersion so only the new pod's
// Added event arrives.
package main

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("watch2")
	defer cancel()

	// Three pods that already exist before we start watching.
	for _, name := range []string{"old-1", "old-2", "old-3"} {
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, name), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", name, err)
		}
	}

	list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		exkit.Failf("listing pods: %v", err)
	}

	watcher, err := cs.CoreV1().Pods(ns).Watch(ctx, metav1.ListOptions{
		ResourceVersion: list.ResourceVersion,
	})
	if err != nil {
		exkit.Failf("starting watch: %v", err)
	}
	defer watcher.Stop()

	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "new-pod"), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating new pod: %v", err)
		}
	}()

	// The FIRST Added event must be the new pod — if old-1/old-2/old-3 are
	// replayed here, the watch started from the wrong resourceVersion.
	timeout := time.After(30 * time.Second)
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				exkit.Failf("watch channel closed unexpectedly")
			}
			if event.Type != watch.Added {
				continue // status updates on the new pod are fine
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				exkit.Failf("unexpected object type %T on the watch channel", event.Object)
			}
			exkit.AssertEqual("first Added event after the list", pod.Name, "new-pod")
			exkit.Successf("watch started at the list's resourceVersion — no replayed history, just news")
			return
		case <-timeout:
			exkit.Failf("timed out waiting for the new pod's Added event")
		}
	}
}
