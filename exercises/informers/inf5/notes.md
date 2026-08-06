## inf5 — indexing the cache

```go
err := informer.AddIndexers(cache.Indexers{
	"byTier": func(obj any) ([]string, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil, nil
		}
		return []string{pod.Labels["tier"]}, nil
	},
})
// … Start, WaitForCacheSync …
frontend, err := informer.GetIndexer().ByIndex("byTier", "frontend")
```

**Why it works**

- The informer's store is a `cache.Indexer` — a keyed store (inf3) *plus* any
  number of secondary indexes. `cache.Indexers` maps an index **name** to an
  `IndexFunc`, and the informer maintains the reverse map as objects are added,
  updated and deleted.
- `ByIndex(name, value)` is then a map lookup instead of a scan. For a
  controller doing "find every pod owned by this ReplicaSet" on every reconcile,
  that is the difference between O(1) and O(all pods in the cache), on the
  hottest path it has.
- The `IndexFunc` returns a **slice** because one object can belong to several
  buckets — return two strings and the pod is filed under both. Returning `nil`
  files it under nothing, which is the clean way to skip objects the index
  doesn't apply to.

**Key detail:** the value you file under has to be the value you look up. The
broken version files pods under `pod.Name` and then asks for `"frontend"`,
which is nobody's name — so `ByIndex` returns an empty slice and a `nil` error.
Same quiet-miss shape as a wrong label selector or a wrong cache key: **check
the length, not just the error.**

Ordering matters too. `AddIndexers` must be called **before** the informer
starts; afterwards it returns
`informer has already started, can not add indexers`. The objects already in the
cache were never filed, so the index would be silently incomplete — refusing is
the right call, and it is the same "materialise everything before `Start`"
discipline as inf4.

The convention for index names is a plain lowercase string, and client-go ships
one for the common case: `cache.NamespaceIndex` (`"namespace"`), backed by
`cache.MetaNamespaceIndexFunc`. controller-runtime exposes the same machinery
through `mgr.GetFieldIndexer().IndexField(...)`, which is what makes
`client.MatchingFields{...}` work in a reconciler — the *same* mechanism, one
layer up.

One caution carried over from inf1: objects returned by `ByIndex` are pointers
into the shared cache. `DeepCopy()` before mutating.

**References**

- `cache.Indexer`: https://pkg.go.dev/k8s.io/client-go/tools/cache#Indexer
- `cache.Indexers` / `IndexFunc`: https://pkg.go.dev/k8s.io/client-go/tools/cache#IndexFunc
- controller-runtime `FieldIndexer`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#FieldIndexer
