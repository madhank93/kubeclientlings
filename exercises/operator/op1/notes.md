## op1 — the Reconcile contract

```go
var cm corev1.ConfigMap
if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil // gone — nothing left to reconcile
	}
	return ctrl.Result{}, err
}
```

**Why it works**

- controller-runtime's manager owns everything you assembled by hand in the
  `controllers` topic: the informers, the cache, the workqueue, the workers, the
  rate limiter. You write one method.
- `ctrl.Request` carries a `types.NamespacedName` — the same
  `namespace/name` key from inf3, already parsed. Rebuilding it by hand
  (`types.NamespacedName{Name: req.Name}`) drops the namespace, and a Get with
  an empty namespace looks for a cluster-scoped object that doesn't exist.
- The `(ctrl.Result, error)` return is the whole control surface:
  - `{}, nil` — done, don't come back until something changes.
  - `{}, err` — failed; requeue with exponential backoff (wq2, for free).
  - `{RequeueAfter: d}, nil` — come back in `d` regardless. For polling
    external state.

**Under the hood**

- The manager builds a cache-backed `client.Client`: reads go to an informer
  cache keyed by GVK, writes go straight to the apiserver. `req.NamespacedName`
  is the same `namespace/name` the workqueue carried, already split.
- The returned `ctrl.Result` is fed back into the controller's rate limiter — an
  error requeues with exponential backoff, `RequeueAfter` schedules an
  `AddAfter`, and a zero result forgets the key.

**Common mistake**

- Returning the `IsNotFound` error instead of swallowing it. The manager
  requeues an object that will never exist, so the controller spins at backoff
  rate forever with nothing visible but a climbing error counter.

**Key detail:** `IsNotFound` is a **success**, not an error. The object was
deleted; the cache already forgot it; there is nothing to reconcile. Returning
the error instead makes the manager retry forever on an object that will never
exist — a hot loop with backoff, visible only as rising error counters.
`client.IgnoreNotFound(err)` is the idiomatic one-liner for this (op2 uses it).

Also note what this Reconcile does **not** do: it never looks at the *event*. It
receives a name, reads current state and makes it correct. That's
level-triggered reconciliation, and it's why the same function handles create,
update, resync and startup without knowing which it is.

Two supporting details in this code:

- The **patch** (`client.MergeFrom(cm.DeepCopy())`) rather than `Update`: no
  `resourceVersion`, so a second event arriving mid-reconcile can't 409. The
  `DeepCopy` is essential — `cm` comes from the shared cache and the patch is
  computed as the diff between the copy and your mutations.
- The early return when the label is already set makes the reconcile a **no-op**
  when nothing needs doing. Without it, the patch triggers an update event which
  triggers another reconcile — a self-sustaining loop.

**See also:** op2 (wiring child objects into the same reconcile) · op3 (converging rather
than reacting) · ctrl2 (the same loop, hand-wired) · pods5 (`IsNotFound` as a
success signal) · the [operator chapter](../README.md)

**References**

- controller-runtime: https://pkg.go.dev/sigs.k8s.io/controller-runtime
- `reconcile.Reconciler`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
- Kubebuilder book: https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html
