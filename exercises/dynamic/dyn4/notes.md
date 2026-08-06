## dyn4 — asking the server what it serves

```go
resources, err := disco.ServerResourcesForGroupVersion("apps/v1")
for _, r := range resources.APIResources {
	if r.Name == "deployments" { ... r.Kind, r.Namespaced, r.Verbs }
}
```

**Why it works**

- Every API server publishes a machine-readable index of itself:
  `/api` and `/apis` list group/versions, and `/apis/apps/v1` lists that
  version's resources. The discovery client wraps those endpoints.
- Each `metav1.APIResource` carries what a generic client needs: `Name` (the
  plural path segment), `Kind`, `Namespaced`, `Verbs` (which of
  get/list/watch/create/update/patch/delete are supported), `ShortNames`
  (`deploy`), and `Group`/`Version` for resources promoted from another group.
- This is how `kubectl` works at all — `kubectl get deploy` is a discovery
  lookup that resolves `deploy` → `apps/v1` → `deployments`, and it's how
  `kubectl api-resources` prints its table.

**Key detail:** asking for a group/version the server doesn't serve is an
**error**, not an empty list. `apps/v1beta1` was removed in Kubernetes 1.16, so
hardcoding it fails outright — which is the good outcome. The failure mode to
fear is the opposite: code that assumes a resource exists and gets a confusing
404 much later.

Useful entry points:

- `ServerGroups()` — every group and its versions, including the preferred one.
- `ServerPreferredResources()` — one entry per resource at the server's
  preferred version. This is what you want when scanning for "everything".
- `ServerVersion()` — the cluster's Kubernetes version.
- `IsResourceEnabled` / checking `Verbs` — feature-detect before calling.

Discovery is chatty (one request per group/version), so client-go ships
`memory.NewMemCacheClient` and a disk-backed cached client; `RESTMapper` is
usually built on top of one of those. Note the API also supports **aggregated
discovery** on modern servers, which returns the whole tree in one request.

Never hardcode a groupVersion for a resource that might move — feature-detect
with discovery instead, and your client keeps working across cluster versions.

**References**

- Discovery client: https://pkg.go.dev/k8s.io/client-go/discovery
- API groups: https://kubernetes.io/docs/reference/using-api/#api-groups
- Deprecated API migration guide: https://kubernetes.io/docs/reference/using-api/deprecation-guide/
