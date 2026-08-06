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

**References**

- Dynamic client: https://pkg.go.dev/k8s.io/client-go/dynamic
- `schema.GroupVersionResource`: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
- API terminology: https://kubernetes.io/docs/reference/using-api/api-concepts/#standard-api-terminology
