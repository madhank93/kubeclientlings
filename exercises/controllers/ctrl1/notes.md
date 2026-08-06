## ctrl1 — the Get/Done contract

```go
item, shutdown := q.Get() // marks the item as being processed
...
q.Done(item)              // releases it; a copy re-added meanwhile is now deliverable
```

**Why it works**

The queue keeps three internal collections, and the contract falls out of how
they interact:

- `queue` — the ordered list of keys waiting.
- `dirty` — the set of keys that *should* be processed. `Add` puts a key here.
- `processing` — keys currently in flight, handed out by `Get`.

`Add(key)`: if the key is already in `dirty`, do nothing (that's the dedup). If
it's in `processing`, mark it dirty but **do not** put it back on the queue —
that's the hold-back. `Get`: pop from `queue`, move to `processing`.
`Done(key)`: remove from `processing`, and if it became dirty meanwhile, *now*
append it to `queue`.

So the sequence in this exercise — `Get("a")`, `Add("a")`, `Done("a")` — yields
`a, b, a`. Without the `Done`, `"a"` sits in `processing` forever, the re-added
copy is never released, and the third `Get` blocks until the test times out.

**Key detail:** this is what makes concurrent workers safe. A key is in
`processing` at most once, so N workers can share one queue and no two ever
reconcile the same object at the same time — no locking in your reconcile code.
It's also what collapses a storm: an object updated fifty times during one slow
reconcile produces exactly **one** more delivery, not fifty.

The consequence is that missing `Done` doesn't crash, log or block anything
visible. It silently removes one object from your controller's attention
forever. Hence the rule: `defer q.Done(key)` on the line after `Get`, always,
so a panic in reconcile can't strand the key either.

Note the shutdown protocol too: `q.Get()` returns `(zero, true)` once `ShutDown`
has been called and the queue is drained — that boolean is the worker's exit
signal, not an error.

**References**

- `workqueue.Interface`: https://pkg.go.dev/k8s.io/client-go/util/workqueue#TypedInterface
- Queue implementation: https://github.com/kubernetes/client-go/blob/master/util/workqueue/queue.go
- sample-controller: https://github.com/kubernetes/sample-controller/blob/master/controller.go
