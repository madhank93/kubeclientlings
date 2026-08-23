## dyn3 — a CRD only serves the versions it declares

```go
// the CRD above declares exactly one version: v1alpha1, served=true, storage=true
gvr := schema.GroupVersionResource{Group: "kubeclientlings.dev", Version: "v1alpha1", Resource: "widgets"}

widget := &unstructured.Unstructured{Object: map[string]any{
	"apiVersion": "kubeclientlings.dev/v1alpha1", // must agree with the GVR
	"kind":       "Widget",
	...
}}
```

**Why it works**

- A CRD registers new endpoints on the running API server. `spec.group` +
  `spec.names.plural` + each entry in `spec.versions` become the paths
  `/apis/kubeclientlings.dev/<version>/namespaces/<ns>/widgets`. Only versions
  with `served: true` get a path at all.
- Asking for `v1` gives a 404 that reads exactly like "this resource doesn't
  exist" — the server cannot distinguish "wrong version" from "no such
  resource", so the error message never points at the real cause. Read the CRD's
  `spec.versions` when a custom resource seems to have vanished.
- `apiVersion` inside the object must match the GVR you send it to. They are
  validated against each other, and a mismatch is a 400.
- The dynamic client is the natural fit here: there are no generated Go structs
  for a type invented at runtime, so the object is built as a `map[string]any`.

**Under the hood**

- Creating a CRD writes an object to `apiextensions.k8s.io/v1`; a controller
  inside the apiserver then builds the CRD's structural schema, installs
  handlers for each served version and refreshes the discovery document. Only
  then does it set `Established=True`.
- Pruning happens on write against that structural schema, before validation, so
  a field the schema does not declare is removed rather than rejected.

**Common mistake**

- Creating a custom resource immediately after creating its CRD. The endpoints
  are not registered yet, so you get a 404 that reads exactly like a typo — and
  it fails often enough to be maddening and rarely enough to reach production.
  Poll `Established`.

**Key detail:** creating a CRD is asynchronous. The object exists immediately,
but the API server has to register handlers and refresh discovery before the new
endpoints answer. That is what the `Established` condition means, and polling for
it — as this exercise does — is mandatory. Creating a custom resource straight
after creating its CRD is a race that fails often enough to be maddening and
rarely enough to reach production.

Exactly one version carries `storage: true` — that's the one etcd persists;
others are served through conversion. `spec.versions[].schema` is a structural
OpenAPI v3 schema and is **required** in `apiextensions.k8s.io/v1`; it's what
validates and prunes unknown fields. Try setting a field the schema doesn't
declare and watch it silently disappear.

Finally, CRDs are cluster-scoped and shared. That's why the exercise deletes any
leftover from a previous run before creating its own — and why deleting a CRD
deletes every custom resource of that type, cluster-wide, with no confirmation.

**See also:** dyn1 (the GVR shape) · sub2 (adding the status subresource to this CRD) ·
crd1 (typed access to what you just created) · the
[dynamic chapter](../README.md)

**References**

- Custom resources: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
- CRD versioning: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/
- `apiextensionsv1` types: https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1
