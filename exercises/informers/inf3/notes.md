## inf3 — cache keys are "namespace/name"

```go
key, err := cache.MetaNamespaceKeyFunc(created) // "my-ns/target"
obj, exists, err := informer.GetStore().GetByKey(key)
```

**Why it works**

- The informer's store is a map keyed by a string. `MetaNamespaceKeyFunc` is the
  default `KeyFunc`: it returns `"<namespace>/<name>"` for a namespaced object
  and just `"<name>"` for a cluster-scoped one (no leading slash).
- `GetByKey` returns `(obj, exists, err)`. A bare `"target"` isn't the key the
  store used, so you get `exists == false` and **no error** — the same quiet
  miss shape as `unstructured.Nested*` and label selectors. Check the boolean.
- Using the function instead of `fmt.Sprintf("%s/%s", ns, name)` matters because
  it handles the cluster-scoped case, and because it's the *same* function the
  informer used when it stored the object. Two different key formats in one
  program is a bug you will not enjoy finding.

**Under the hood**

- `MetaNamespaceKeyFunc` calls `meta.Accessor(obj)` and formats
  `namespace + "/" + name`, omitting the separator entirely when the namespace
  is empty. `SplitMetaNamespaceKey` inverts it by counting slashes, which is why
  a key with two is an error rather than a namespace containing a slash.
- The `Indexer` stores under whatever `KeyFunc` the informer was built with, so
  a hand-built key that differs by even a separator addresses a different slot.

**Common mistake**

- Looking up by name alone for a namespaced object. `GetByKey` returns
  `exists == false` with a nil error, which reads as "it was deleted" — so a
  reconcile takes the cleanup branch on an object that is very much still there.

**Key detail:** this string is the universal currency of controllers. Event
handlers compute it and push it on a workqueue; the reconcile loop pops it and
splits it back with `cache.SplitMetaNamespaceKey(key)` → `(namespace, name,
err)`. That's why controller-runtime's `Reconcile` takes a
`types.NamespacedName` and nothing else.

Passing a *key* rather than the object is a deliberate design choice, not a
micro-optimisation: keys deduplicate in the queue, and by the time you reconcile
you re-read current state instead of acting on a stale snapshot. That is what
"level-triggered" means in practice.

Two nearby APIs:

- `cache.DeletionHandlingMetaNamespaceKeyFunc` unwraps a
  `DeletedFinalStateUnknown` tombstone first — use it in delete handlers.
- The typed **Lister** (`lister.Pods(ns).Get(name)`) is nicer than
  `GetStore().GetByKey` for everyday reads: it does the key building and the
  type assertion for you, and returns a proper `NotFound` error.

**See also:** inf1 (the store being keyed) · ctrl2 (the key as a queue item) · inf5 (the
secondary indexes over the same store) · the [informers chapter](../README.md)

**References**

- `cache.MetaNamespaceKeyFunc`: https://pkg.go.dev/k8s.io/client-go/tools/cache#MetaNamespaceKeyFunc
- `cache.Store` / `Indexer`: https://pkg.go.dev/k8s.io/client-go/tools/cache#Indexer
- `types.NamespacedName`: https://pkg.go.dev/k8s.io/apimachinery/pkg/types#NamespacedName
