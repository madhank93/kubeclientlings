// inf2
//
// Informers PUSH: AddEventHandler registers callbacks for every Add,
// Update and Delete the watch sees. But registering a handler wires
// nothing up by itself — until the factory is STARTED no goroutine runs,
// no watch opens, and no handler ever fires. Forgetting Start() is the
// classic silent informer bug.
//
// Start the factory so the handler sees the pod appear.
package main

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("inf2")
	defer cancel()

	factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
	podInformer := factory.Core().V1().Pods()

	added := make(chan string, 8)
	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if pod, ok := obj.(*corev1.Pod); ok {
				added <- pod.Name
			}
		},
	})
	if err != nil {
		exkit.Failf("adding event handler: %v", err)
	}

	// Handlers registered, cache configured... now actually turn it on.
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced) {
		exkit.Failf("cache never synced")
	}

	if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "pushed"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	select {
	case name := <-added:
		exkit.AssertEqual("pod delivered to the Add handler", name, "pushed")
	case <-time.After(30 * time.Second):
		exkit.Failf("no Add event after 30s — did the informer machinery actually start?")
	}

	exkit.Successf("the informer pushed the event to you — no polling anywhere")
}
