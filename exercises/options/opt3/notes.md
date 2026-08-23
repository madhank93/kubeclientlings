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

**Under the hood**

- Apply is a `PATCH` with content type `application/apply-patch+yaml`. The
  server converts the incoming partial object into a typed field set, merges it
  with `structured-merge-diff`, and rewrites `metadata.managedFields` with one
  entry per manager naming the exact field paths it owns.
- Ownership is per *leaf path*, not per object, which is what lets two managers
  hold different keys inside the same map without either noticing the other.

**Common mistake**

- Sending a plain `corev1.Pod` instead of an apply configuration. Go zero values
  are indistinguishable from unset in that type, so the server reads them as a
  claim over every field — and your manager quietly becomes the owner of things
  it never meant to touch.

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

**See also:** ssa1 (the same mechanism, in depth) · ssa2 (what happens when two managers
disagree) · pods4 (the patch types apply sits alongside) · the
[options chapter](../README.md)

**References**

- Server-side apply: https://kubernetes.io/docs/reference/using-api/server-side-apply/
- Apply configurations: https://pkg.go.dev/k8s.io/client-go/applyconfigurations
- `metav1.ApplyOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ApplyOptions
