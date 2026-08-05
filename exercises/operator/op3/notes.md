## op3 — everything desired goes inside mutate()

```go
_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
	// runs on BOTH paths — cm holds the LIVE object here on the update path
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["replicas"] = strconv.Itoa(int(*dep.Spec.Replicas))
	return controllerutil.SetControllerReference(&dep, cm, r.scheme)
})
```

**Why it works**

`CreateOrUpdate` does exactly this, in this order:

1. `Get` the object by the key in `cm`'s ObjectMeta, **into `cm`**.
2. If `NotFound`: run `mutate()`, then `Create`. Return `OperationResultCreated`.
3. Otherwise: snapshot the fetched object, run `mutate()`, and `Update` **only
   if `mutate` actually changed something**. Return `OperationResultUpdated` or
   `OperationResultNone`.

Step 1 is the trap. That `Get` **overwrites** `cm` with the live object, so any
desired state you set before the call is destroyed on the update path. The
create path still works — nothing was found, so nothing overwrote it — which is
why the bug ships: creation looks perfect and the object simply never updates
again. Drift is never repaired and a scale-up is never mirrored.

Putting the mutations inside means they are applied *on top of* whatever the
server currently has, which is what "lay desired state over observed state"
means.

**Key detail:** three rules for a `mutate` function.

- **Idempotent.** It runs on every reconcile, and its no-change case is what
  produces `OperationResultNone` and skips the write entirely. Without that, the
  update triggers an event, which triggers a reconcile, which updates again —
  the classic hot loop.
- **Never change the object's name, namespace, or anything immutable.** The key
  was used to fetch; changing it makes the subsequent write nonsense.
- **Set the ownerRef inside too.** It's part of desired state, and the child
  needs it for the `Owns()` routing that drives drift repair (op2).

Only touch fields you own. Wiping `cm.Data` wholesale instead of setting one key
would delete anything another writer added. This is the same discipline as
Server-Side Apply, minus the server-side bookkeeping — and `CreateOrPatch` is
the sibling that sends a patch rather than a full update, which is gentler on
objects with several writers.

The three phases the exercise walks through — create, update, drift repair — all
come out of this one code path. That's the payoff: a reconciler that converges
rather than one that reacts.

**References**

- `controllerutil.CreateOrUpdate`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil#CreateOrUpdate
- Kubebuilder book: https://book.kubebuilder.io/
- Operator pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
