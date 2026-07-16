// inf3
//
// Objects in an informer cache are indexed by KEY, and for namespaced
// objects the key is "namespace/name" — that's what MetaNamespaceKeyFunc
// produces and what every controller's workqueue carries around. Ask the
// store for a bare name and it politely finds nothing: no error, just
// exists=false.
//
// Look the pod up by its real cache key.
package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("inf3")
	defer cancel()

	created, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "target"), metav1.CreateOptions{})
	if err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
	// Materialize the informer BEFORE Start — the factory only starts
	// informers that already exist when Start is called.
	informer := factory.Core().V1().Pods().Informer()
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		exkit.Failf("cache never synced")
	}

	// MetaNamespaceKeyFunc builds the canonical key: "namespace/name".
	key, err := cache.MetaNamespaceKeyFunc(created)
	if err != nil {
		exkit.Failf("building cache key: %v", err)
	}

	obj, exists, err := informer.GetStore().GetByKey(key)
	if err != nil {
		exkit.Failf("reading from the store: %v", err)
	}

	exkit.AssertTrue("pod found in the cache under key "+key, exists)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		exkit.Failf("unexpected object type %T in the store", obj)
	}
	exkit.AssertEqual("the cached pod", pod.Name, "target")
	exkit.Successf("keys are namespace/name — the same string controllers push through workqueues")
}
