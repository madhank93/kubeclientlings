// watch3
//
// Raw watches die: API servers close them, networks drop them, timeouts
// fire. RetryWatcher re-establishes the stream automatically — but it
// refuses to guess where to start. It demands a real resourceVersion
// ("" and "0" are rejected), which forces you into the correct
// list-then-watch pattern.
//
// Give the RetryWatcher a real starting point.
//
// I AM NOT DONE
package main

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("watch3")
	defer cancel()

	// "0" is exactly what RetryWatcher refuses — it can't retry from a
	// made-up point in history. It needs a REAL resourceVersion, and a
	// List is where you get one.
	watcher, err := watchtools.NewRetryWatcherWithContext(ctx, "0", &cache.ListWatch{
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			return cs.CoreV1().Pods(ns).Watch(ctx, options)
		},
	})
	if err != nil {
		exkit.Failf("creating RetryWatcher: %v", err)
	}
	defer watcher.Stop()

	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "resilient"), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating pod: %v", err)
		}
	}()

	timeout := time.After(30 * time.Second)
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				exkit.Failf("watch channel closed unexpectedly")
			}
			if event.Type != watch.Added {
				continue
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				exkit.Failf("unexpected object type %T", event.Object)
			}
			exkit.AssertEqual("pod seen through the RetryWatcher", pod.Name, "resilient")
			exkit.Successf("RetryWatcher running from a real resourceVersion — it survives dropped connections")
			return
		case <-timeout:
			exkit.Failf("timed out waiting for the pod's Added event")
		}
	}
}
