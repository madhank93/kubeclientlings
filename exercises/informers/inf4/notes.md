## inf4 — scope the watch, not the read

```go
factory := informers.NewSharedInformerFactoryWithOptions(cs, 0, informers.WithNamespace(ns))
```

**Why it works**

- `NewSharedInformerFactory(cs, 0)` builds informers whose List+Watch is
  cluster-wide. The cache then holds every pod in every namespace —
  `kube-system`, other tenants, everything — and the API server maintains a
  watch stream feeding you all of it.
- `WithNamespace(ns)` is applied to the underlying `ListWatch`, so the narrowing
  happens **server-side**: only your namespace's objects are listed, watched,
  transferred and stored. Filtering after the fact in Go would save none of
  that.
- The cost is real. A pod is a few KB of Go structs; a busy cluster has tens of
  thousands. Cluster-scoped informers in a controller that only cares about one
  namespace are a routine cause of multi-gigabyte controller memory and of
  API-server watch load that shows up as everyone else's latency.
- Note the ordering comment in the code: `factory.Start` only starts informers
  that **already exist**. Calling `factory.Core().V1().Pods().Informer()` after
  `Start` creates one that never runs — another silent hang. Materialise every
  informer first, then `Start`, then `WaitForCacheSync`.

**Key detail:** "Shared" is the other half of the efficiency story. The factory
returns *the same* informer for the same GVR, so ten controllers asking for pods
share one watch and one cache. That only holds within a factory — build a second
factory and you get a second watch. One factory per process (or per scope) is
the rule.

Other narrowing options, which compose:

- `informers.WithTweakListOptions(func(o *metav1.ListOptions) { o.LabelSelector = "app=web" })`
  — narrow by label or field selector. Combined with `WithNamespace`, this is how
  you get a cache holding *only* what you reconcile.
- `informers.WithTransform(...)` — strip fields you never read (managedFields,
  annotations) before the object enters the cache. Large clusters win a lot here.

The trade-off to be aware of: a scoped informer's lister can only answer
questions about what it watched. Ask it about another namespace and you get an
empty result, not an error — same quiet miss as everywhere else in this topic.

**References**

- `informers` options: https://pkg.go.dev/k8s.io/client-go/informers#SharedInformerOption
- `SharedIndexInformer`: https://pkg.go.dev/k8s.io/client-go/tools/cache#SharedIndexInformer
- Controller architecture: https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md
