## ssa1 — apply is declarative, and declarations need an author

```go
apply := applyconfigcorev1.ConfigMap("settings", ns).
	WithData(map[string]string{"replicas": "3"})

_, err := cs.CoreV1().ConfigMaps(ns).Apply(ctx, apply, metav1.ApplyOptions{
	FieldManager: "kubeclientlings",
})
```

**Why it works**

- Server-Side Apply reverses the usual flow. Instead of "read the object,
  change it, write it all back", you send **only the fields you intend to own**
  and the API server merges them into whatever is there.
- It records ownership per field in `metadata.managedFields`. That ledger is the
  entire feature — without a `FieldManager` name there is nothing to record
  ownership under, so the server rejects the request rather than guessing.
- The apply-configuration builders (`applyconfigcorev1.ConfigMap(...).WithData(...)`)
  produce structs where **every field is a pointer**, so unset and zero are
  distinguishable. That precision is required: a plain `corev1.ConfigMap` full of
  Go zero values would read as a claim over every field in the type.
- One HTTP call, no `Get` first, no `resourceVersion`, no 409 retry loop. Compare
  the read-modify-write cycle in pods3.

**Key detail:** apply also **removes**. If your manager set a field last time and
your next apply omits it, the server sees you released it and deletes the value.
That is what makes apply genuinely declarative — and why an apply must always
send the complete set of fields you own, never a partial diff. Send half your
config and you just deleted the other half.

The corollary: use a **stable** manager name. A name derived from the pod name,
a UUID or a timestamp creates a new owner on every restart, leaving the old
name's fields orphaned in `managedFields` forever.

Apply is also the answer for CRDs, where strategic merge patch does not work
(deploy3): the server reads list-merge semantics from the CRD's OpenAPI schema
instead of Go struct tags.

**References**

- Server-Side Apply: https://kubernetes.io/docs/reference/using-api/server-side-apply/
- Apply configurations: https://pkg.go.dev/k8s.io/client-go/applyconfigurations
- `metav1.ApplyOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ApplyOptions
