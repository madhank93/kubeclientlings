// inf5
//
// A lister can answer "everything" and "everything matching a label", but both
// walk the whole cache. When a controller needs "every pod for THIS owner" on
// every reconcile, that linear scan is the hot path. Indexers fix it: you
// register a function that maps an object to the keys it should file under,
// and the cache maintains a reverse map you can hit directly.
//
// The catch: an index is only as good as the key its function returns. Return
// the wrong string and ByIndex finds nothing — no error, just an empty slice.
//
// Index the pods by their tier label.
package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("inf5")
	defer cancel()

	fixtures := []struct{ name, tier string }{
		{"web-1", "frontend"},
		{"web-2", "frontend"},
		{"db-1", "backend"},
	}
	for _, f := range fixtures {
		pod := exkit.NginxPod(ns, f.name)
		pod.Labels = map[string]string{"tier": f.tier}
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", f.name, err)
		}
	}

	factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
	informer := factory.Core().V1().Pods().Informer()

	// AddIndexers must happen BEFORE Start — an informer that is already
	// running refuses new indexers, because the objects already in the cache
	// were never filed under them.
	err := informer.AddIndexers(cache.Indexers{
		"byTier": func(obj any) ([]string, error) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return nil, nil
			}
			// File each pod under its tier. The return is a SLICE because one
			// object may belong to many buckets — returning two strings here
			// would file the pod under both. Returning nil (or an empty
			// slice) files it under nothing, which is how you skip objects
			// the index does not apply to.
			return []string{pod.Labels["tier"]}, nil
		},
	})
	if err != nil {
		exkit.Failf("adding the indexer: %v", err)
	}

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		exkit.Failf("cache never synced")
	}

	frontend, err := informer.GetIndexer().ByIndex("byTier", "frontend")
	if err != nil {
		exkit.Failf("querying the index: %v", err)
	}
	backend, err := informer.GetIndexer().ByIndex("byTier", "backend")
	if err != nil {
		exkit.Failf("querying the index: %v", err)
	}

	exkit.AssertEqual("pods indexed under tier=frontend", len(frontend), 2)
	exkit.AssertEqual("pods indexed under tier=backend", len(backend), 1)
	exkit.Successf("O(1) lookup straight out of the cache — no scan, no API call")
}
