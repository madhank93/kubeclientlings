## crd2 — TypeMeta is what the dynamic client routes on

```go
w := Widget{
	TypeMeta:   metav1.TypeMeta{APIVersion: "kubeclientlings.dev/v1alpha1", Kind: "Widget"},
	ObjectMeta: metav1.ObjectMeta{Name: "w1"},
	Spec:       WidgetSpec{Size: 7},
}

m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&w)
u := &unstructured.Unstructured{Object: m}
```

**Why it works**

- `ToUnstructured` is the mirror of crd1: struct → `map[string]any`, matched by
  `json` tags. It takes a **pointer** (it reflects over an addressable value)
  and returns the map, which you wrap in an `unstructured.Unstructured` to get
  the accessor methods.
- `metav1.TypeMeta` embeds with `json:",inline"`, so setting it puts
  `apiVersion` and `kind` at the top level of the resulting map. Leave it zero
  and — because `TypeMeta`'s fields carry `omitempty` — those keys simply don't
  appear. `u.GetKind()` returns `""`, and the dynamic client has nothing to
  route on.
- The dynamic client is schema-blind by design: it does not know what a Widget
  is. `apiVersion` + `kind` in the body are how the server validates the object
  against the endpoint you POSTed to, so both have to be there and both have to
  agree with the GVR.

**Key detail:** this is the asymmetry that catches people. With the **typed**
clientset you never set `TypeMeta` — the client knows the type at compile time
and fills it in, and objects you `Get` back often have it *cleared*. With the
**dynamic** client you must always set it yourself. So a struct that round-trips
fine through the typed path silently loses its identity on the dynamic path.

Where does the GVK come from in real code? From a `runtime.Scheme`:

```go
gvks, _, err := scheme.ObjectKinds(&w)
u.SetGroupVersionKind(gvks[0])
```

That's what controller-runtime does internally, and it beats hardcoding the
strings in two places.

Note also that `ToUnstructured` produces only JSON-compatible values
(`string`, `bool`, `int64`, `float64`, `map[string]any`, `[]any`). A struct
field of a type it can't represent is an error, not a silent drop — the one
place this direction is louder than the other.

**References**

- `runtime.UnstructuredConverter`: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#UnstructuredConverter
- `metav1.TypeMeta`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#TypeMeta
- `runtime.Scheme`: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Scheme
