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
  anything else immediately. `retry.DefaultRetry` is a short backoff (5 steps,
  ~10ms base) sized for exactly this.

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

**References**

- `retry.RetryOnConflict`: https://pkg.go.dev/k8s.io/client-go/util/retry#RetryOnConflict
- Resource versions: https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions
- Concurrency control: https://kubernetes.io/docs/reference/using-api/api-concepts/#optimistic-concurrency-control
