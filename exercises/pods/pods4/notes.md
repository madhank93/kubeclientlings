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

**Under the hood**

- The patch type becomes the request's `Content-Type`
  (`application/strategic-merge-patch+json`, `application/merge-patch+json`,
  `application/json-patch+json`, `application/apply-patch+yaml`), and the
  apiserver dispatches on that header alone.
- For a strategic merge the server loads the built-in Go type's struct tags,
  applies the merge against the stored object and writes the result in one
  transaction — so there is no window for another writer, which is what removes
  the need for a `resourceVersion`.

**Common mistake**

- Declaring `JSONPatchType` over an object-shaped body (or the reverse). The
  server tries to decode an array and finds an object, so you get a 4xx that
  talks about decoding rather than about patch types — the header and the bytes
  have to agree.

**Key detail:** merging is why the `app` label survives. A strategic merge
patch touches only the keys you named; `metadata.labels` is a map, so `tier` is
added and `app` is left alone. Compare with the pods3 bug, where a PUT of a
partial object wipes everything you omitted.

To *delete* a key with a merge patch, set it to `null`:
`{"metadata":{"labels":{"tier":null}}}`.

**See also:** pods3 (the read-modify-write this replaces) · deploy3 (merge keys, when the
patch touches a list) · ssa1 (apply, the type that also tracks ownership) · the
[pods chapter](../README.md)

**References**

- Update API objects in place: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/
- `types` patch constants: https://pkg.go.dev/k8s.io/apimachinery/pkg/types#PatchType
- JSON Merge Patch (RFC 7386): https://datatracker.ietf.org/doc/html/rfc7386
