// inf1
//
// Informers replace polling: List once, Watch forever, keep a local cache
// you read for free. But the cache starts EMPTY — Start() only launches
// the machinery in the background. Read before the initial sync finishes
// and you get whatever happens to be there: usually nothing.
//
// Wait for the cache to sync before reading from the lister.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("inf1")
	defer cancel()

	for _, name := range []string{"cached-1", "cached-2", "cached-3"} {
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, name), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", name, err)
		}
	}

	factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
	podInformer := factory.Core().V1().Pods()
	lister := podInformer.Lister()

	factory.Start(ctx.Done())

	// Start() returns immediately — the initial List is still in flight on
	// a goroutine somewhere. Reading the lister RIGHT NOW races it (and
	// loses). Something needs to block here until the cache has synced.

	pods, err := lister.Pods(ns).List(labels.Everything())
	if err != nil {
		exkit.Failf("listing from the cache: %v", err)
	}

	exkit.AssertEqual("pods in the informer cache", len(pods), 3)
	exkit.Successf("cache synced — the lister answered from memory, no API round-trip")
}
