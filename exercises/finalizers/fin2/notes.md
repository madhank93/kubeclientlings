## fin2 — removing the last finalizer is what completes the delete

```go
got, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "guarded", metav1.GetOptions{})
got.Finalizers = nil // cleanup is done — release it
_, err = cs.CoreV1().ConfigMaps(ns).Update(ctx, got, metav1.UpdateOptions{})
```

**Why it works**

- The object is already terminating: `deletionTimestamp` is set, and the API
  server is waiting for one thing — an empty `metadata.finalizers`.
- Clearing the list and `Update`ing it is a normal write to metadata. On seeing
  the last finalizer go, the server completes the pending deletion immediately.
  You never call `Delete` again; the delete was already requested and queued.
- Several controllers can each hold their own finalizer on one object. The
  object goes away when the **last** one is removed, so each controller removes
  only its own string — never `= nil` in production, which would silently
  discard other controllers' claims and delete the object before their cleanup
  ran.

**Under the hood**

- The update path checks `deletionTimestamp != nil && len(finalizers) == 0` and,
  when both hold, performs the delete inside the same transaction — so the
  object is gone by the time your `Update` returns.
- A terminating object still accepts metadata writes, which is what makes this
  possible; most controllers additionally refuse spec changes on an object whose
  `deletionTimestamp` is set.

**Common mistake**

- `obj.Finalizers = nil` in a controller. It discards every other controller's
  claim as well as your own, so the object is deleted before their cleanup ran —
  and they will never be told. Remove only your own string.

**Key detail:** this is the second half of a two-branch reconcile that every
finalizer-using controller has:

```go
if obj.DeletionTimestamp.IsZero() {
	// normal path: make sure our finalizer is present, then reconcile
	if !controllerutil.ContainsFinalizer(obj, myFinalizer) {
		controllerutil.AddFinalizer(obj, myFinalizer)
		return r.Update(ctx, obj)
	}
	return r.reconcileNormal(ctx, obj)
}

// deletion path
if controllerutil.ContainsFinalizer(obj, myFinalizer) {
	if err := r.cleanupExternalResources(ctx, obj); err != nil {
		return err // keep the finalizer; retry
	}
	controllerutil.RemoveFinalizer(obj, myFinalizer)
	return r.Update(ctx, obj)
}
return nil
```

Two properties that fall out of it and matter:

- **Cleanup must be idempotent.** The reconcile may run many times against a
  terminating object; deleting an already-deleted cloud resource has to be a
  no-op, not an error.
- **Only remove the finalizer after cleanup succeeds.** Returning the error
  keeps the finalizer, so the object stays and you get retried. Remove it first
  and the object vanishes along with any record of what you failed to clean up —
  a leak with nothing left to point at it.

Note also: a terminating object is immutable in `spec` for most controllers'
purposes, and adding a *new* finalizer to an object that already has a
`deletionTimestamp` is rejected — the window has closed.

**See also:** fin1 (adding the finalizer) · op2 (`controllerutil`, which has the
add/remove/contains helpers) · pods3 (the conflict retry these metadata writes
need) · the [finalizers chapter](../README.md)

**References**

- Finalizers: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
- `controllerutil.RemoveFinalizer`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil#RemoveFinalizer
- Operator patterns: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
