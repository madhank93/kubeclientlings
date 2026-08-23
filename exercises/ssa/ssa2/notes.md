## ssa2 — conflicts are the feature; Force is the override

```go
// mgr-a owns data.owner
cs.CoreV1().ConfigMaps(ns).Apply(ctx, applyA, metav1.ApplyOptions{FieldManager: "mgr-a"})

// mgr-b takes it, knowingly
cs.CoreV1().ConfigMaps(ns).Apply(ctx, applyB, metav1.ApplyOptions{FieldManager: "mgr-b", Force: true})
```

**Why it works**

- Once `mgr-a` applies `data.owner`, the server records that field under
  `mgr-a` in `managedFields`. When `mgr-b` applies a **different value** to the
  same field, the server returns `409 Conflict` with a body listing exactly
  which fields are contested and who holds them.
- That refusal is the point. Two controllers fighting over one field is the
  classic Kubernetes failure — each overwriting the other every few seconds,
  visible only as an object that flaps. Under `Update`/`Patch` this happens
  silently; under apply the second writer gets an error naming the first.
- `Force: true` says "I know, take it anyway". The field's value changes **and**
  its ownership moves to `mgr-b`; `mgr-a` loses it. If `mgr-a` applies again it
  will get the conflict.
- Applying the *same* value to a field someone else owns is **not** a conflict —
  it becomes co-ownership. Conflicts are about disagreement, not about touching.

**Under the hood**

- `managedFields` holds one entry per (manager, operation, subresource) with the
  owned paths encoded as a `FieldsV1` blob. A conflict is computed by
  intersecting your incoming set with every *other* manager's set and keeping
  the paths where the values disagree.
- The 409 body is a `metav1.Status` whose `details.causes` lists each contested
  path and its current owner, so the error is machine-readable, not just
  human-readable.

**Common mistake**

- Setting `Force: true` everywhere to make conflicts go away. In a controller
  that is correct; in an interactive tool it silently steals fields another
  controller is maintaining, and the controller takes them straight back — an
  invisible tug-of-war.

**Key detail:** when to force, and when not to.

- **Controllers should force.** A controller is the authority over the fields it
  manages; on conflict it should reassert them, or it will wedge on the first
  human `kubectl edit`. `kubectl apply --server-side --force-conflicts` is the
  same escape hatch for people.
- **Interactive tools should not.** The conflict error names the other manager,
  which is precisely the information a human needs before clobbering something a
  controller is maintaining.

To *release* a field rather than steal it, simply stop including it in your
apply — ownership disappears with the value. And note that dropping a manager
name entirely (renaming your controller, say) does not clean up: those fields
stay owned by a name nobody uses again.

**See also:** ssa1 (ownership, and how apply removes fields) · opt3 (`ApplyOptions`) ·
op3 (the same "only touch what you own" discipline, client-side) · the
[SSA chapter](../README.md)

**References**

- Conflicts: https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts
- Field management: https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management
- `metav1.ApplyOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ApplyOptions
