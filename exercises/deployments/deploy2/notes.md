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

**References**

- `ptr` helpers: https://pkg.go.dev/k8s.io/utils/ptr
- `DeploymentSpec`: https://pkg.go.dev/k8s.io/api/apps/v1#DeploymentSpec
- `UpdateScale`: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
