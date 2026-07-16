// inf1
//
// Informers replace polling: List once, Watch forever, keep a local cache
// you read for free. But the cache starts EMPTY — Start() only launches
// the machinery in the background. Read before the initial sync finishes
// and you get whatever happens to be there: usually nothing.
//
// Wait for the cache to sync before reading from the lister.
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/madhank93/kubeclientlings/internal/exkit"
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

	// Block until the informer's initial List has landed in the cache.
	// Every controller you will ever read does exactly this before serving.
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced) {
		exkit.Failf("cache never synced")
	}

	pods, err := lister.Pods(ns).List(labels.Everything())
	if err != nil {
		exkit.Failf("listing from the cache: %v", err)
	}

	exkit.AssertEqual("pods in the informer cache", len(pods), 3)
	exkit.Successf("cache synced — the lister answered from memory, no API round-trip")
}
