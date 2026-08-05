## pods4 — the patch type must match the patch body

```go
patch := []byte(`{"metadata":{"labels":{"tier":"frontend"}}}`)
_, err := cs.CoreV1().Pods(ns).Patch(ctx, "hello", types.StrategicMergePatchType, patch, metav1.PatchOptions{})
```

**Why it works**

- `Patch` is a PATCH request: the body describes a *change*, not a whole
  object, so there is no `resourceVersion` and no 409 to retry. One round-trip,
  no read.
- The **type** argument sets the `Content-Type` header and tells the server how
  to interpret the bytes. The four types:
  - `types.StrategicMergePatchType` — Kubernetes' own. Body is shaped like the
    object; maps merge; **lists merge by a patch-merge key** (containers by
    `name`, ports by `containerPort`) instead of being replaced. Only works on
    built-in types, which carry the `patchStrategy` struct tags.
  - `types.MergePatchType` — RFC 7386. Same object-shaped body, but lists are
    replaced wholesale. This is what you use for CRDs.
  - `types.JSONPatchType` — RFC 6902. Body is an **array** of operations:
    `[{"op":"add","path":"/metadata/labels/tier","value":"frontend"}]`.
  - `types.ApplyPatchType` — Server-Side Apply (see the `ssa` topic).
- Here the body is object-shaped, so the type has to be a merge type. Declaring
  `JSONPatchType` over an object-shaped body makes the server fail to decode it
  — the mismatch is a 4xx, not a silent no-op.

**Key detail:** merging is why the `app` label survives. A strategic merge
patch touches only the keys you named; `metadata.labels` is a map, so `tier` is
added and `app` is left alone. Compare with the pods3 bug, where a PUT of a
partial object wipes everything you omitted.

To *delete* a key with a merge patch, set it to `null`:
`{"metadata":{"labels":{"tier":null}}}`.

**References**

- Update API objects in place: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/
- `types` patch constants: https://pkg.go.dev/k8s.io/apimachinery/pkg/types#PatchType
- JSON Merge Patch (RFC 7386): https://datatracker.ietf.org/doc/html/rfc7386
