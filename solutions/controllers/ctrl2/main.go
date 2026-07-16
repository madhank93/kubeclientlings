// ctrl2
//
// This is a whole controller in one file: informer → workqueue → worker.
// The glue is the KEY. Handlers push "namespace/name" strings (never whole
// objects — the cache dedupes by key), and the worker splits the key back
// apart to act. Push the wrong string and the worker reconciles a pod that
// doesn't exist.
//
// Enqueue proper cache keys so the worker can find the pod.
package main

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("ctrl2")
	defer cancel()

	q := workqueue.NewTyped[string]()
	defer q.ShutDown()

	factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
	informer := factory.Core().V1().Pods().Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			// MetaNamespaceKeyFunc builds "namespace/name" — the one true
			// currency between informers and workers.
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			q.Add(key)
		},
	})
	if err != nil {
		exkit.Failf("adding event handler: %v", err)
	}

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		exkit.Failf("cache never synced")
	}

	if _, err := cs.CoreV1().Pods(ns).Create(ctx, exkit.NginxPod(ns, "managed"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating pod: %v", err)
	}

	// The worker: pop one key, split it, mark the pod reconciled.
	key, shutdown := q.Get()
	if shutdown {
		exkit.Failf("queue shut down before the worker got a key")
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		exkit.Failf("splitting key %q: %v", key, err)
	}

	patch, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{"labels": map[string]string{"reconciled": "true"}},
	})
	if _, err := cs.CoreV1().Pods(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		exkit.Failf("reconciling %q: %v", key, err)
	}
	q.Done(key)

	exkit.WaitFor(ctx, "the pod to carry the reconciled label", func(ctx context.Context) (bool, error) {
		pod, err := cs.CoreV1().Pods(ns).Get(ctx, "managed", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Labels["reconciled"] == "true", nil
	})

	exkit.Successf("informer → key → queue → worker → patch: you just wrote a controller")
}
