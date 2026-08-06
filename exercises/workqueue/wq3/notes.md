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

**Key detail:** this three-branch shape belongs *around* the reconcile, not
inside it:

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

Worth knowing where this pattern does and doesn't appear upstream.
`sample-controller` **does not** cap retries at all — it only does
`AddRateLimited` on error and `Forget` on success, so a poison item there
retries forever (slowly). The capped form is what the real controllers in
`kubernetes/kubernetes` use, and both `deployment` and `endpointslice` set:

```go
maxRetries = 15
```

15 attempts under the default limiter (5ms doubling to a 1000s ceiling) spans
several minutes of retrying before giving up. The `3` in this exercise keeps
the run fast; pick your own from how transient your failures really are, and
lean higher than feels natural — dropping a key is permanent until something
else touches the object.

**References**

- `workqueue`: https://pkg.go.dev/k8s.io/client-go/util/workqueue
- sample-controller's loop (no cap — compare): https://github.com/kubernetes/sample-controller/blob/master/controller.go
- `maxRetries = 15` in the Deployment controller: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/deployment_controller.go
- `utilruntime.HandleError`: https://pkg.go.dev/k8s.io/apimachinery/pkg/util/runtime#HandleError
