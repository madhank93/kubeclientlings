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

**Under the hood**

- The handler runs on the informer's delivery goroutine and the worker on its
  own, connected only by the queue. That separation is what lets the informer
  keep draining its watch while a slow reconcile is in progress.
- `Patch` bypasses optimistic concurrency entirely: no `resourceVersion` on the
  wire means the apiserver merges against whatever is current, so a second event
  arriving mid-reconcile cannot produce a 409.

**Common mistake**

- Enqueueing `pod.Name` instead of the namespaced key. `SplitMetaNamespaceKey`
  then yields an empty namespace, and `Pods("")` builds a cluster-wide URL that
  a namespaced write cannot use — so the reconcile fails on every object, in a
  way that reads like a permissions problem.

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

**See also:** inf3 (the key function) · ctrl1 (the `Get`/`Done` contract the worker obeys)
· wq3 (what to do when the reconcile keeps failing) · op1 (the same loop, with
the machinery hidden) · the [controllers chapter](../README.md)

**References**

- sample-controller: https://github.com/kubernetes/sample-controller
- `cache.MetaNamespaceKeyFunc` / `SplitMetaNamespaceKey`: https://pkg.go.dev/k8s.io/client-go/tools/cache
- Controllers concept: https://kubernetes.io/docs/concepts/architecture/controller/
