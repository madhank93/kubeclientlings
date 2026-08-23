## pods3 — read-modify-write under optimistic concurrency

```go
err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
	current, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
	if err != nil {
		return err
	}
	current.Labels["tier"] = "frontend"
	_, err = cs.CoreV1().Pods(ns).Update(ctx, current, metav1.UpdateOptions{})
	return err
})
```

**Why it works**

- `Update` is a **PUT**: it replaces the whole object. The object you send must
  carry the `resourceVersion` you read, and the server accepts the write only
  if that version is still current. Different version → `409 Conflict`. This is
  optimistic concurrency control, and it is why you must `Get` first.
- The `Get` inside the closure is the important part. On a conflict the retry
  re-runs the whole body, so it re-reads a **fresh** object with the new
  `resourceVersion`, re-applies the mutation and tries again. Hoisting the
  `Get` outside the closure would retry with the same stale version forever.
- `retry.RetryOnConflict` retries only on `IsConflict` errors and returns
  anything else immediately. `retry.DefaultRetry` is
  `wait.Backoff{Steps: 5, Duration: 10ms, Factor: 1.0, Jitter: 0.1}` — note
  `Factor: 1.0`, so it does **not** grow: five attempts roughly 10ms apart with
  a little jitter. That is deliberate. A write conflict clears as soon as the
  other writer finishes, so retrying promptly beats backing off; the growing
  backoff in the `workqueue` topic solves a different problem.

**Under the hood**

- `resourceVersion` is etcd's revision for that key, passed through opaquely.
  The registry does a transactional compare-and-swap on it, so a mismatch
  becomes `StatusReasonConflict` and never a partial write.
- `RetryOnConflict` is `wait.ExponentialBackoff` with a predicate: it runs the
  closure, returns immediately unless `apierrors.IsConflict(err)`, and otherwise
  sleeps the next backoff step and runs it again. Nothing is retried that is not
  a conflict.

**Common mistake**

- Hoisting the `Get` out of the closure "to avoid re-reading". Every retry then
  resends the same stale `resourceVersion`, so all five attempts fail
  identically and the error you finally return is a conflict that could never
  have cleared.

**Key detail:** for Pods the conflict is not theoretical — the kubelet writes
`status` continuously, so the resourceVersion of a running pod changes on its
own every few seconds. Any read-modify-write on a live object needs the retry.

Sending a hand-built object with no `resourceVersion` doesn't fail: an empty
version means "I don't care, overwrite unconditionally", so the server accepts
it and **replaces** the object — silently discarding the spec, the status and
every field you didn't include. That is the destructive version of this bug,
and it is much worse than a 409.

If you only want to change a couple of fields, `Patch` (pods4) or Server-Side
Apply (the `ssa` topic) avoid the whole cycle.

**See also:** pods4 (the patch that avoids this cycle entirely) · ssa1 (the declarative
version) · sub2 (status writes, the most contended of all) · the
[pods chapter](../README.md)

**References**

- `retry.RetryOnConflict`: https://pkg.go.dev/k8s.io/client-go/util/retry#RetryOnConflict
- Resource versions: https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions
- HTTP PUT, conflicts and dropped fields: https://kubernetes.io/docs/reference/using-api/api-concepts/#update-mechanism-update
