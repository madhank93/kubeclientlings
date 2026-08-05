## pods5 — NotFound is the success signal

```go
_, err := cs.CoreV1().Pods(ns).Get(ctx, "doomed", metav1.GetOptions{})
if apierrors.IsNotFound(err) {
	return true, nil // gone — this is success, not an error
}
if err != nil {
	return false, err // a real error (network, auth, ...) aborts the wait
}
return false, nil // still terminating
```

**Why it works**

- `Delete` is asynchronous. It sets `metadata.deletionTimestamp` and returns;
  the object stays visible while finalizers run and the kubelet actually stops
  the containers. Anything that needs the object *gone* has to poll.
- `apierrors.IsNotFound(err)` unwraps the error to a `*errors.StatusError` and
  checks for HTTP 404 / reason `NotFound`. That is far more robust than string
  matching, and it is `nil`-safe — `IsNotFound(nil)` is `false`, so the
  ordering above is correct.
- Separating "the thing I wanted happened" from "something broke" is the whole
  point: returning the NotFound as an error would abort the poll at the exact
  moment it succeeded.

**Key detail:** the same family covers the rest of the API's error vocabulary,
and controllers lean on all of it: `IsAlreadyExists` (create-if-absent),
`IsConflict` (retry the read-modify-write), `IsForbidden` (RBAC — never worth
retrying), `IsTooManyRequests` / `IsServerTimeout` (back off and retry). Use
`apierrors.ReasonForError(err)` when you need to switch on more than one.

Two flags on `DeleteOptions` are worth knowing: `PropagationPolicy`
(`Foreground` waits for dependents and keeps the owner around until they are
gone, `Background` deletes the owner immediately, `Orphan` leaves dependents
behind) and `Preconditions` (refuse the delete unless the UID/resourceVersion
still matches — protection against deleting a *recreated* object with the same
name).

**References**

- `apierrors` helpers: https://pkg.go.dev/k8s.io/apimachinery/pkg/api/errors
- Garbage collection & propagation: https://kubernetes.io/docs/concepts/architecture/garbage-collection/
- `metav1.DeleteOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#DeleteOptions
