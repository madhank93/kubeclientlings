## ctrl2 — a whole controller: informer → key → queue → worker

```go
AddFunc: func(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj) // "namespace/name"
	if err != nil {
		return
	}
	q.Add(key)
},
...
namespace, name, err := cache.SplitMetaNamespaceKey(key) // back apart
_, err = cs.CoreV1().Pods(namespace).Patch(ctx, name, ...)
```

**Why it works**

- The handler's job is tiny: turn the object into a key and enqueue it. All the
  real work happens on the worker side, off the informer's goroutine.
- `MetaNamespaceKeyFunc` builds `"namespace/name"`;
  `SplitMetaNamespaceKey` reverses it. Using the pair keeps the format in one
  place and handles the cluster-scoped case (no slash) correctly.
- A bare `pod.Name` splits into `namespace: ""`, and `Pods("")` builds a
  cluster-wide URL that a namespaced `Patch` cannot use. The request doesn't go
  to the "default" namespace — it fails, or worse, addresses something you
  didn't mean.

**Key detail:** enqueueing a *key* rather than the *object* is the design
decision that defines Kubernetes controllers. Keys deduplicate in the queue, and
by the time the worker runs it re-reads current state — so it acts on what is
true **now**, not on a snapshot from whenever the event fired. That is
level-triggered reconciliation, and it's why controllers survive missed events,
restarts and out-of-order delivery.

Everything else in this file is the standard skeleton, in the order that matters:

1. Build the informer(s) — **before** `Start`.
2. Register handlers — also before `Start`.
3. `factory.Start(stopCh)`, then `cache.WaitForCacheSync(...)`.
4. Run N workers, each looping `Get` → reconcile → `Done`/`Forget`/`AddRateLimited`.

The reconcile here patches rather than read-modify-writes: no `resourceVersion`,
no 409, no retry loop (pods4). Real controllers do the same wherever they can,
and reach for Server-Side Apply when they need to own a whole block of fields.

One thing this file skips for brevity: a real worker must handle the key's
object being **gone**. `lister.Pods(ns).Get(name)` returning `IsNotFound` means
"it was deleted" — that's a normal reconcile outcome (clean up, return nil), not
an error to retry.

**References**

- sample-controller: https://github.com/kubernetes/sample-controller
- `cache.MetaNamespaceKeyFunc` / `SplitMetaNamespaceKey`: https://pkg.go.dev/k8s.io/client-go/tools/cache
- Controllers concept: https://kubernetes.io/docs/concepts/architecture/controller/
