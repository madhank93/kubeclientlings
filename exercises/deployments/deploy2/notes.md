## deploy2 — why so many Kubernetes fields are pointers

```go
replicas := int32(3) // int32, not int — the field is *int32
dep.Spec.Replicas = &replicas
```

**Why it works**

- `Spec.Replicas` is `*int32`. Go has no implicit numeric conversion, so `&x`
  where `x` is an `int` is a `*int` and simply will not assign — a compile
  error, not a runtime surprise. Declaring `int32(3)` fixes the type; the
  address-of then produces the `*int32` the field wants.
- It's a **pointer** so the API can distinguish three states that a plain
  `int32` collapses into one: unset (`nil` → the server applies the default of
  1), explicitly zero (`*p == 0` → scale to nothing, a real and useful
  request), and any other value. Without the pointer, `omitempty` on a plain
  `int32` would drop a deliberate 0 from the JSON.
- The same reasoning explains the pointer soup elsewhere in the API:
  `TerminationGracePeriodSeconds`, `RunAsNonRoot`, `Optional` on env-var
  sources. Any field where "not specified" and "the zero value" mean different
  things.

**Under the hood**

- Every optional field in the API is generated from a `// +optional` marker,
  which makes the Go field a pointer with `json:",omitempty"`. That pairing is
  what lets the JSON round-trip distinguish absent from zero — an `omitempty`
  non-pointer would erase a deliberate `0`.
- On the server, `nil` is filled in by the defaulting pass
  (`SetDefaults_Deployment` sets one replica) before validation ever sees the
  object, which is why a Deployment created with no replicas reads back as 1.

**Common mistake**

- `dep.Spec.Replicas = &replicas` where `replicas` is an `int`. It does not
  compile, which is the good case. The bad case is reaching for
  `*dep.Spec.Replicas` on a freshly built object and panicking on the nil
  pointer — read it with `ptr.Deref(dep.Spec.Replicas, 1)`.

**Key detail:** you can't take the address of a literal (`&3` and
`&int32(3)` are both invalid), which is why the named-variable dance exists.
Rather than writing it out every time, use the generic helpers:

```go
import "k8s.io/utils/ptr"
dep.Spec.Replicas = ptr.To[int32](3)
value := ptr.Deref(dep.Spec.Replicas, 1) // nil-safe read with a default
```

(`ptr.To` supersedes the older `k8s.io/utils/pointer.Int32`.)

For scaling specifically, the whole-object Update here is the teaching version;
production code usually uses the **scale subresource** —
`cs.AppsV1().Deployments(ns).UpdateScale(...)` or a patch against `/scale` —
which touches only the replica count and can't clobber anything else.

**See also:** sub1 (the scale subresource, the production way to do this) · deploy4 (what
the replica counters mean afterwards) · deploy1 (the object being scaled) · the
[deployments chapter](../README.md)

**References**

- `ptr` helpers: https://pkg.go.dev/k8s.io/utils/ptr
- `DeploymentSpec`: https://pkg.go.dev/k8s.io/api/apps/v1#DeploymentSpec
- `UpdateScale`: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
