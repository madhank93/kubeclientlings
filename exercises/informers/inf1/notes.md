## inf1 — the cache starts empty

```go
factory.Start(ctx.Done()) // launches goroutines, returns immediately

if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced) {
	exkit.Failf("cache never synced")
}

pods, err := lister.Pods(ns).List(labels.Everything()) // now safe
```

**Why it works**

- An informer is list-then-watch (watch2) with a cache bolted on. Internally a
  `Reflector` does the initial `List`, pushes the objects into a `DeltaFIFO`, and
  a loop pops them into the `Indexer` (the thread-safe store); then it `Watch`es
  and keeps applying deltas forever.
- `Start` spawns those goroutines and returns **immediately**. At that instant
  the store is empty, so a read races the initial List and reliably loses.
- `HasSynced` flips true once every object from the initial List has been
  *processed* — not merely received. `WaitForCacheSync` polls it (and returns
  `false` if the stop channel closes first, which is why the return value must
  be checked rather than ignored).
- After the sync, `lister.Pods(ns).List(...)` reads from local memory: no HTTP,
  no serialisation, microseconds. That is the entire point of the machinery —
  a controller reconciling thousands of times a second never touches the API
  server to *read*.

**Key detail:** `WaitForCacheSync` gives you a **consistent starting point, not
freshness**. Once synced, the cache is eventually-consistent: it trails the API
server by however long a watch event takes to arrive. Never do read-modify-write
straight off a lister — `Get` from the API for the live `resourceVersion`, or be
ready for the `409` (pods3).

Two more rules worth burning in:

- **Never mutate an object from a lister.** It's a pointer into the shared
  cache; mutating it corrupts what every other consumer sees. `DeepCopy()`
  first.
- The `0` in `NewSharedInformerFactory(cs, 0)` is the **resync period**, not a
  poll interval. A resync re-delivers everything already in the cache as
  synthetic Update events; it does not re-fetch from the server. `0` disables
  it, which is what modern controllers want — level-triggered reconciliation
  makes the safety net unnecessary.

**References**

- `informers` package: https://pkg.go.dev/k8s.io/client-go/informers
- `cache.WaitForCacheSync`: https://pkg.go.dev/k8s.io/client-go/tools/cache#WaitForCacheSync
- Controller architecture: https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md
