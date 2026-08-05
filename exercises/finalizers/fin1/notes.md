## fin1 — a finalizer turns Delete into "mark terminating"

```go
cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
	Name:       "guarded",
	Namespace:  ns,
	Finalizers: []string{finalizer}, // "kubeclientlings.dev/protect"
}}
```

**Why it works**

- `metadata.finalizers` is a list of opaque strings. The API server's only rule
  is: **if the list is non-empty, do not remove the object.**
- A `Delete` on such an object sets `metadata.deletionTimestamp` and returns
  success. The object stays fully readable and listable, now in the
  "terminating" state. `Get` returns it; `IsNotFound` is false.
- Each string is a claim by some controller that it has work to do first. The
  server never interprets them — it just counts them.
- This is what lets an operator deallocate external state (a cloud load
  balancer, a database, a DNS record) *before* its Kubernetes object vanishes.
  Without it, the object disappears the instant the user hits delete and the
  controller never learns what it was supposed to clean up.

**Key detail:** finalizers are the reason objects get "stuck terminating"
forever, and it is one of the most common Kubernetes support questions. If the
controller that owns a finalizer is uninstalled, crashed, or never existed,
nothing will ever remove the string — and the object cannot be deleted, not with
`--force`, not with `--grace-period=0`. The only escape is to edit
`metadata.finalizers` to empty by hand, which is exactly what the cleanup at the
end of this exercise does. (A namespace stuck `Terminating` is nearly always
this, one level down.)

Conventions worth following:

- **Namespace your finalizer**: `yourdomain.com/name`. Anything with a
  `kubernetes.io` or `k8s.io` prefix is reserved, and bare names are rejected on
  many resources.
- **Add it at create time or on first reconcile**, before you allocate anything
  external. Add it after and you have a window where a delete loses the object.
- Adding a finalizer is a normal metadata write, so it competes with other
  writers — `controllerutil.AddFinalizer` + `Update` inside a
  `retry.RetryOnConflict` is the safe form.

**References**

- Finalizers: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
- Garbage collection: https://kubernetes.io/docs/concepts/architecture/garbage-collection/
- `controllerutil` helpers: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil
