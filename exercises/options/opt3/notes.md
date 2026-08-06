## opt3 — apply needs an identity

```go
applied, err := cs.CoreV1().Pods(ns).Apply(ctx, pod, metav1.ApplyOptions{
	FieldManager: "kubeclientlings", // mandatory: who owns these fields
})
```

**Why it works**

- Server-side apply records, per field, **which manager set it**. That
  bookkeeping lives in `metadata.managedFields`, and it is what lets two
  controllers edit the same object safely: each owns its own fields, and the
  server can tell when one tries to take a field the other holds.
- Because ownership is the entire point, an apply with no manager name is
  meaningless and the server rejects it — `FieldManager` is required on
  `ApplyOptions` in a way it isn't on `UpdateOptions` (where client-go derives
  a default from the user agent).
- The **apply configuration** types (`corev1apply.Pod(...).WithLabels(...)`) are
  generated structs where every field is a pointer, so "not set" is
  representable. That matters: with a plain `corev1.Pod` you cannot express
  "I don't care about replicas" versus "replicas should be 0", and apply would
  read your zero values as a claim of ownership over those fields.

**Key detail:** field ownership is also how apply **removes** things. If your
manager previously set a field and your next apply omits it, the server sees
that you released it and deletes the value. That's declarative behaviour you do
not get from `Patch`, and it's the reason an apply must always send the
*complete* set of fields you intend to own — never a partial diff.

Use a stable, specific manager name (your controller's name), not a per-process
or per-run string: a new name on every restart accumulates junk in
`managedFields` and orphans the fields the old name owned.

Conflicts: if another manager owns a field you're setting, the apply fails with
a 409 listing the conflicts. `Force: true` in `ApplyOptions` takes ownership
anyway — appropriate for a controller that is the authority on those fields,
and a footgun everywhere else.

**References**

- Server-side apply: https://kubernetes.io/docs/reference/using-api/server-side-apply/
- Apply configurations: https://pkg.go.dev/k8s.io/client-go/applyconfigurations
- `metav1.ApplyOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ApplyOptions
