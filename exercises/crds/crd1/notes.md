## crd1 — the converter maps by json tag

```go
type WidgetSpec struct {
	Size int64 `json:"size"` // must equal the key in the unstructured map
}

var w Widget
err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &w)
```

**Why it works**

- `DefaultUnstructuredConverter` walks the `map[string]any` and your struct in
  parallel, matching by **`json` struct tag** — the same rules `encoding/json`
  uses, minus the actual serialisation round-trip (it's reflection over the map,
  which is why it's faster than marshal-then-unmarshal).
- `json:"sizes"` against a map key of `"size"` finds no match, so `Size` keeps
  its zero value. And that is the whole hazard: **no error is returned**. A
  mismatched tag is not a decode failure, it's a missing field, exactly as a
  JSON document without that key would be.
- `metav1.TypeMeta` is embedded with `json:",inline"`, so `apiVersion` and
  `kind` land on the outer object rather than nesting — that's how `w.Kind`
  ends up populated from the top-level `"kind"` key.
- The type of the map value matters: JSON numbers are `int64` here, and the
  converter will error if the map holds an `int` where the struct wants `int64`.
  Same `int64` rule as the `unstructured.Nested*` helpers.

**Under the hood**

- `FromUnstructured` uses a cached per-type reflection plan: for each struct
  field it records the `json` tag name and a conversion function, then walks the
  map once. No marshal, no unmarshal, no intermediate `[]byte`.
- Absent keys are simply skipped, which is how the "no error on a wrong tag"
  behaviour arises — the converter cannot distinguish a typo from an optional
  field the document omitted.

**Common mistake**

- Trusting the conversion because it returned `nil`. A misspelled tag, a field
  with no tag at all, or a `json:"-"` leaves the Go field at its zero value and
  reports nothing. Assert on a value you expect, not on the error.

**Key detail:** this conversion is the seam between the dynamic client and typed
code. `dyn.Resource(gvr).Get(...)` returns `*unstructured.Unstructured`;
`FromUnstructured(u.Object, &w)` gives you a struct with compile-time field
names, IDE completion and no `map[string]any` in your business logic. It's the
pragmatic middle ground between generated clients (which need code generation
per CRD) and raw maps everywhere.

Because it's the same mechanism, all the ordinary `encoding/json` tag rules
apply: `omitempty`, `-` to skip, and — worth knowing — a field with **no** tag
at all is matched by its Go field name, capital letter included, which almost
never matches a Kubernetes key.

**See also:** crd2 (the reverse direction, which is louder) · dyn2 (reading the same map
without a struct) · dyn1 (where the unstructured object came from) · the
[CRDs chapter](../README.md)

**References**

- `runtime.UnstructuredConverter`: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#UnstructuredConverter
- `encoding/json` tag rules: https://pkg.go.dev/encoding/json#Marshal
- Custom resources: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
