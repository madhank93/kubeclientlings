## dyn1 — Kind names objects, Resource names URLs

```go
gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
list, err := dyn.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
```

**Why it works**

- The typed clientset knows every path at compile time. The dynamic client
  knows nothing, so **you** name the endpoint — that is what a
  `GroupVersionResource` is: the three path components of a Kubernetes URL.
- `Resource` is the lowercase plural **path segment**, the thing you see in
  `/api/v1/namespaces/<ns>/pods`. It is *not* the Kind. `Kind` (`Pod`,
  `Deployment`) is the CamelCase name that appears **inside** objects, in
  `kind:` and in Go type names. Same concept, two namespaces, and mixing them up
  produces a 404 on a URL nobody serves.
- `Group: ""` is the **core** group. Core resources live under `/api/v1`;
  everything else lives under `/apis/<group>/<version>`. So `apps/v1` is
  `{Group: "apps", Version: "v1", Resource: "deployments"}`.
- Results are `*unstructured.UnstructuredList` — `map[string]any` under the
  hood, with accessors like `GetKind()`, `GetName()`, `GetLabels()`. No
  generated structs, which is exactly why this client works on CRDs it has
  never heard of.

**Under the hood**

- The dynamic client builds the URL by hand from the GVR:
  `/apis/{group}/{version}` (or `/api/{version}` when the group is empty), then
  `namespaces/{ns}` if a namespace was set, then the resource segment. There is
  no scheme lookup and no type registry involved.
- Responses are decoded by `unstructured.UnstructuredJSONScheme` straight into
  `map[string]any`, which is why the client works on types it has never been
  compiled against.

**Common mistake**

- Putting the Kind in the `Resource` field — `Pod` instead of `pods`. The
  resulting URL is one nobody serves, so you get a 404 that reads like the
  object is missing rather than like the path is wrong.

**Key detail:** the plural is not always mechanical. `endpoints` is already
plural, `networkpolicies` isn't `networkpolicys`, and subresources get their own
segments. Rather than guessing, resolve at runtime with a **RESTMapper**:

```go
mapping, err := mapper.RESTMapping(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "v1")
gvr := mapping.Resource
```

That's how `kubectl` turns `deploy` on the command line into
`apps/v1/deployments`, and it's the right tool whenever the type isn't known
until runtime.

Note also `dyn.Resource(gvr)` with no `.Namespace(ns)` — that's the
cluster-scoped (or all-namespaces) form, the equivalent of `Pods("")`.

**See also:** dyn2 (reading what comes back) · dyn4 (resolving GVRs at runtime instead of
guessing) · crd2 (the `TypeMeta` the dynamic client routes on) · pods1 (the same
URL, via the typed client) · the [dynamic chapter](../README.md)

**References**

- Dynamic client: https://pkg.go.dev/k8s.io/client-go/dynamic
- `schema.GroupVersionResource`: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
- API terminology: https://kubernetes.io/docs/reference/using-api/api-concepts/#standard-api-terminology
