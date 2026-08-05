## dyn2 — walking unstructured objects safely

```go
msg, found, err := unstructured.NestedString(u.Object, "data", "message")
```

**Why it works**

- `u.Object` is a `map[string]any` mirroring the object's JSON exactly. Reaching
  `data.message` by hand means two type assertions and two nil checks, and one
  wrong key panics.
- `NestedString(obj, path...)` walks the path for you and returns
  `(value, found, err)`:
  - `found == false` — nothing at that path. **Not an error.** Optional fields
    are normal, so this is the common case, not a failure.
  - `err != nil` — something *is* there but it's the wrong type (you asked for
    a string and found a map). That's a real bug, in your code or in the object.
- The path is every key from the top of the object down. A ConfigMap has no
  top-level `message` — the value nests under `data`, so `"data", "message"` is
  the full path. Getting this wrong yields `found=false`, which is why you must
  actually check it rather than using the value.

**Key detail:** the family is `NestedString`, `NestedBool`, `NestedInt64`,
`NestedFloat64`, `NestedStringMap`, `NestedStringSlice`, `NestedSlice`,
`NestedMap` and `NestedFieldNoCopy`. Note the **Int64** — JSON has one numeric
type, so every integer in an unstructured object is an `int64`. Asking for an
`int` or `int32` is a type error, and it's the single most common surprise here.

`NestedMap` / `NestedSlice` **deep-copy** what they return, which is what you
want when the object came from an informer cache: mutating a cached object
corrupts the cache for every other consumer. `NestedFieldNoCopy` skips the copy
when you only need to read and the cost matters.

Writing back is `unstructured.SetNestedField(u.Object, value, "spec", "size")`,
and it will error unless the value is a JSON-compatible type (`string`, `bool`,
`int64`, `float64`, `map[string]any`, `[]any`).

**References**

- `unstructured` package: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
- `Unstructured` accessors: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured
- Dynamic client: https://pkg.go.dev/k8s.io/client-go/dynamic
