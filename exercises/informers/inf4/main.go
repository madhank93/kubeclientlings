// inf4
//
// A factory built with no options watches the WHOLE cluster — kube-system,
// other tenants, everything. That's cache memory and API load you pay for
// objects you will never touch. WithNamespace scopes the underlying
// List+Watch so only your namespace ever crosses the wire.
//
// Scope the informer so the cache holds exactly this exercise's pods.
//
// I AM NOT DONE
package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("inf4")
	defer cancel()

	for _, name := range []string{"mine-1", "mine-2", "mine-3"} {
		if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, name), metav1.CreateOptions{}); err != nil {
			exkit.Failf("creating %s: %v", name, err)
		}
	}

	// No options = this informer Lists and Watches pods in EVERY
	// namespace. The count below includes kube-system and friends, and
	// the assertion wants only this exercise's three pods. Scope the
	// factory to the namespace `ns`.
	factory := informers.NewSharedInformerFactory(cs, 0)
	podInformer := factory.Core().V1().Pods()
	// Touch Informer() BEFORE Start — the factory only starts informers
	// that already exist when Start is called.
	informer := podInformer.Informer()
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		exkit.Failf("cache never synced")
	}

	pods, err := podInformer.Lister().List(labels.Everything())
	if err != nil {
		exkit.Failf("listing from the cache: %v", err)
	}

	exkit.AssertEqual("pods the informer cache knows about", len(pods), 3)
	exkit.Successf("the informer only watched %s — small cache, cheap watch, happy API server", ns)
}
