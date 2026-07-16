// watch1
//
// Watch() streams changes as they happen. Every event on the channel has a
// TYPE — Added, Modified, Deleted — and reacting to the wrong type is one
// of the quietest bugs in Kubernetes tooling: the code compiles, runs, and
// waits forever.
//
// Observe the pod being created and then deleted.
//
// I AM NOT DONE
package main

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("watch1")
	defer cancel()

	watcher, err := cs.CoreV1().Pods(ns).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		exkit.Failf("starting watch: %v", err)
	}
	defer watcher.Stop()

	// Generate one Added and one Deleted event.
	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "watched"), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating pod: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
		// Zero grace period: skip the default 30s graceful shutdown so the
		// Deleted event arrives while we're still watching.
		grace := int64(0)
		if err := cs.CoreV1().Pods(ns).Delete(ctx, "watched", metav1.DeleteOptions{GracePeriodSeconds: &grace}); err != nil {
			exkit.Failf("deleting pod: %v", err)
		}
	}()

	var sawAdded, sawDeleted bool
	timeout := time.After(30 * time.Second)
	for !(sawAdded && sawDeleted) {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				exkit.Failf("watch channel closed before both events were seen (added=%v deleted=%v)", sawAdded, sawDeleted)
			}
			switch event.Type {
			case watch.Added:
				sawAdded = true
			// "DELETE" compiles fine — EventType is just a string — but no
			// event ever carries that value, so this case never fires.
			// The watch package exports the real constants.
			case "DELETE":
				sawDeleted = true
			}
		case <-timeout:
			exkit.Failf("timed out: added=%v deleted=%v — are you checking the right event types?", sawAdded, sawDeleted)
		}
	}

	exkit.AssertTrue("saw the Added event", sawAdded)
	exkit.AssertTrue("saw the Deleted event", sawDeleted)
	exkit.Successf("you watched a pod's whole life go by on one channel")
}
