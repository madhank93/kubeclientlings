## wq1 — Done and Forget are not the same thing

```go
q.Forget("a") // clears the retry history → next failure starts at base delay
```

**Why it works**

The queue tracks two independent things per key, and each has its own call:

- **Processing state** — `Get()` hands you an item and marks it *in flight*;
  `Done(item)` marks it finished. Between those two, re-adding the same key does
  not deliver it again; it is remembered and redelivered *after* `Done`. That's
  what guarantees a controller never processes one key on two workers
  concurrently.
- **Retry history** — `AddRateLimited(item)` asks the rate limiter for a delay
  and increments that key's failure count. `NumRequeues(item)` reads it;
  `Forget(item)` resets it to zero.

`Done` says nothing about retries, so a reconcile that succeeds after three
failures leaves `NumRequeues == 3`. The next failure then starts at
`base·2³` instead of `base` — a controller that gets slower and slower for a
resource that is now healthy, with no obvious cause.

**Under the hood**

- The queue holds three collections: `queue` (ordered, waiting), `dirty` (should
  be processed) and `processing` (in flight). `Add` skips anything already
  dirty; if the key is in `processing` it marks it dirty without queueing, and
  `Done` is what finally appends it.
- The rate limiter is a separate object entirely, keyed by item, with its own
  failure counters — which is precisely why `Done` and `Forget` touch different
  state.

**Common mistake**

- Calling `Done` and treating it as "finished successfully". It only releases
  the in-flight slot; the failure count survives, so the next failure for a key
  that has since recovered starts at whatever backoff it had reached.

**Key detail:** the canonical loop calls **both**, in a fixed shape:

```go
key, shutdown := q.Get()
if shutdown { return }
defer q.Done(key)         // always, even on panic

if err := reconcile(key); err != nil {
	q.AddRateLimited(key) // retry with backoff
	return
}
q.Forget(key)             // success → wipe the history
```

`defer q.Done(key)` is non-negotiable: miss it and the key is stuck "in
flight" forever, so every future add for it is silently swallowed and that
object is never reconciled again.

The rest of the queue's value is deduplication and ordering: it's a set, so
adding the same key ten times before it is processed yields **one** delivery. A
hot object being updated in a loop costs one reconcile, not ten — which is why
event handlers enqueue keys instead of doing work.

The `Typed*` generic constructors are the current API; the old untyped
`workqueue.NewRateLimitingQueue` (items as `any`) is deprecated.

**See also:** wq2 (the counters `Forget` resets) · wq3 (capping them) · ctrl1 (the
in-flight contract in detail) · inf2 (the handler that feeds this queue) · the
[workqueue chapter](../README.md)

**References**

- `workqueue` package: https://pkg.go.dev/k8s.io/client-go/util/workqueue
- `RateLimitingInterface`: https://pkg.go.dev/k8s.io/client-go/util/workqueue#TypedRateLimitingInterface
- sample-controller's loop: https://github.com/kubernetes/sample-controller/blob/master/controller.go
