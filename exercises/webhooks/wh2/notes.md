## wh2 — a patch without a PatchType is silently dropped

```go
patch := []map[string]any{
	{"op": "add", "path": "/metadata/labels", "value": map[string]string{"injected": "true"}},
}
resp.Patch, _ = json.Marshal(patch)

pt := admissionv1.PatchTypeJSONPatch // *PatchType, so it needs a variable
resp.PatchType = &pt
```

**Why it works**

- A mutating webhook doesn't return a modified object — it returns a **patch**
  the API server applies to the incoming object before persisting it.
- `AdmissionResponse.PatchType` is a `*PatchType` and `PatchTypeJSONPatch` is
  the only legal value today. Nil means "no patch here", so the server ignores
  `resp.Patch` entirely: no error, no warning, no mutation. The webhook appears
  to work and does nothing — the same silent-miss shape as a wrong label
  selector or a wrong json tag.
- Because it's a pointer, you cannot take the address of the constant directly;
  hence the `pt` variable. (`ptr.To(admissionv1.PatchTypeJSONPatch)` is the
  tidier form.)
- The patch itself is **RFC 6902 JSON Patch**: an array of
  `{op, path, value}` operations with `/`-separated paths — *not* the
  object-shaped merge patch from pods4. Wrong shape here is a decode error on
  the server side.

**Under the hood**

- The apiserver applies `response.patch` to the incoming object with
  `jsonpatch.DecodePatch(...).Apply(...)` — but only after checking
  `response.patchType`. A nil type short-circuits that whole branch, so the
  bytes are never decoded and never reported on.
- After each mutating webhook the object is re-serialised for the next one,
  which is why mutations must be idempotent under `reinvocationPolicy`.

**Common mistake**

- Marshalling a correct JSON Patch and forgetting `resp.PatchType`. The webhook
  returns 200, the admission chain continues, and the object is stored
  unmodified — the handler looks like it works and does nothing at all.

**Key detail:** `{"op": "add", "path": "/metadata/labels", ...}` **replaces**
the whole labels map if one already exists. To add a single label without
clobbering the others, target the key:

```json
{"op": "add", "path": "/metadata/labels/injected", "value": "true"}
```

…but that fails if `metadata.labels` doesn't exist at all, because JSON Patch
will not create intermediate objects. The robust pattern is to look at the
incoming object and emit either the whole-map op or the single-key op. Note the
escaping rules too: `/` in a key becomes `~1` and `~` becomes `~0`, so
`app.kubernetes.io/name` is `/metadata/labels/app.kubernetes.io~1name`.

Operationally: mutating webhooks run **before** validating ones, and all
mutations complete before any validation — so you can mutate and then validate
the result in the same admission pass. They must be idempotent, because the
server may re-invoke them (up to `reinvocationPolicy`) after other webhooks
mutate the same object.

**See also:** wh1 (the validating half, and the same zero-value trap) · pods4 (the patch
types, including this one) · the [webhooks chapter](../README.md)

**References**

- Mutating webhooks: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-request-and-response
- JSON Patch (RFC 6902): https://datatracker.ietf.org/doc/html/rfc6902
- `admissionv1.AdmissionResponse`: https://pkg.go.dev/k8s.io/api/admission/v1#AdmissionResponse
