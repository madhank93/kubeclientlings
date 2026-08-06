## deploy3 — merge keys identify list elements

```go
patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"web","image":"nginx:1.28-alpine"}]}}}}`)
_, err := cs.AppsV1().Deployments(ns).Patch(ctx, "web", types.StrategicMergePatchType, patch, metav1.PatchOptions{})
```

**Why it works**

- Merging a map is unambiguous — keys identify themselves. Merging a **list**
  is not: is `containers[0]` in your patch the same element as `containers[0]`
  on the server, or a new one to append?
- Strategic merge patch answers that with a **merge key** declared in the Go
  type's struct tags. `PodSpec.Containers` is tagged
  `patchStrategy:"merge" patchMergeKey:"name"`, so the server matches elements
  by `name`: an entry whose name exists is merged into that container, an entry
  with a new name is appended, and everything you didn't mention is untouched.
- Include `"name":"web"` and the server updates that one container's image.
  Omit it and there is nothing to match on, so the patch is rejected.

**Key detail:** merge keys are a property of the **built-in Go types**, which
is why strategic merge patch does not work on CRDs — a
`CustomResourceDefinition` has no `patchStrategy` tags for the server to read.
For custom resources use `types.MergePatchType` (RFC 7386, lists are replaced
wholesale), `types.JSONPatchType` (index-addressed operations), or
Server-Side Apply, which reads list-merge behaviour from the OpenAPI schema and
therefore *does* work on CRDs. That last route is the modern answer, and it's
the `ssa` topic.

Patching `spec.template` changes the pod template hash, which is precisely what
triggers a new ReplicaSet and a rolling update — this patch *is* "deploy a new
version".

**References**

- Strategic merge patch: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
- `PodSpec` struct tags: https://pkg.go.dev/k8s.io/api/core/v1#PodSpec
- kubectl patch task: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/
