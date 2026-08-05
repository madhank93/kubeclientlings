## wq3 — capping retries so poison items get dropped

```go
handleErr := func(key string) {
	if q.NumRequeues(key) < maxRetries {
		q.AddRateLimited(key)
		return
	}
	// Out of retries: give up on this key and clear its history.
	q.Forget(key)
}
```

**Why it works**

- Some failures never resolve: a malformed spec, a reference to a namespace that
  doesn't exist, a bug in your reconcile. Backoff (wq2) slows those down but
  never stops them — capped at `max`, a poison item still comes back forever,
  holding a worker slot and filling your logs.
- `NumRequeues(key)` is the failure counter the rate limiter maintains, so it's
  free to read and needs no state of your own.
- Past the cap you call `Forget` and **do not** re-add. Forgetting matters even
  though you're dropping the key: without it the counter stays high, so if the
  object is later updated and enqueued afresh, its very first failure resumes at
  the maximum backoff.
- Dropping is safe because controllers are level-triggered. The key is gone from
  the queue, but any future change to that object generates a new event and a
  new enqueue with a clean counter. You lose the retry loop, not the object.

**Key detail:** this three-branch shape is the canonical `handleErr` in every
client-go controller, and it belongs *around* the reconcile, not inside it:

```go
func (c *Controller) handleErr(err error, key string) {
	if err == nil {
		c.queue.Forget(key)   // success: reset backoff
		return
	}
	if c.queue.NumRequeues(key) < maxRetries {
		c.queue.AddRateLimited(key) // retry with backoff
		return
	}
	c.queue.Forget(key)             // give up
	utilruntime.HandleError(err)    // and make sure a human can see why
}
```

Dropping silently is the part people get wrong. Emit a Warning Event on the
object (the `watch` topic) or set a `Degraded` condition on its status, so the
failure is visible in `kubectl describe` rather than only in a log line that
scrolled past.

`maxRetries = 3` is `sample-controller`'s value and a reasonable default;
`5` is common too. Match it to how transient your failures actually are.

**References**

- `workqueue`: https://pkg.go.dev/k8s.io/client-go/util/workqueue
- sample-controller `handleErr`: https://github.com/kubernetes/sample-controller/blob/master/controller.go
- `utilruntime.HandleError`: https://pkg.go.dev/k8s.io/apimachinery/pkg/util/runtime#HandleError
