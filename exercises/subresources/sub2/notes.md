## sub2 — status is a separate write path

```go
// on the CRD:
Subresources: &apiextensionsv1.CustomResourceSubresources{
	Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
},

// in the client:
unstructured.SetNestedField(got.Object, "Ready", "status", "phase")
_, err = dyn.Resource(gvr).Namespace(ns).UpdateStatus(ctx, got, metav1.UpdateOptions{})
```

**Why it works**

- Declaring `subresources.status` on the CRD splits one object into two write
  paths: the main endpoint (`PUT .../widgets/first`) and
  `PUT .../widgets/first/status`.
- Once split, each endpoint **ignores** the other's half. The main `Update`
  silently drops any change to `status` — no error, the field just doesn't
  move. `UpdateStatus` symmetrically drops changes to `spec`.
- That silence is the trap and the reason this exercise exists. The write
  succeeds, the response comes back 200, and the value isn't there.
- The split exists so that a user editing `spec` and a controller writing
  `status` cannot clobber each other. Each sends a whole object; each has
  half of it discarded. Without the subresource, a controller's status write
  would also carry — and overwrite — whatever `spec` it last read.

**Key detail:** with the status subresource enabled, `metadata.generation` also
starts behaving properly: it increments **only** on spec changes, never on
status writes. That is what makes the `generation` vs `observedGeneration`
comparison from deploy4 meaningful — without the subresource, a controller
writing status would bump the generation and race itself forever.

RBAC splits too: `update` on `widgets/status` without `update` on `widgets` is
exactly the permission set a controller should hold.

For built-in types the same pattern appears as
`cs.AppsV1().Deployments(ns).UpdateStatus(...)`, and status is enabled on
essentially everything. The pattern for a modern controller is:

```go
obj.Status.Phase = "Ready"
meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{...})
_, err := client.UpdateStatus(ctx, obj, metav1.UpdateOptions{})
```

Status writes hit the same optimistic-concurrency rules as any Update, so wrap
them in `retry.RetryOnConflict` (pods3) — status is the field most likely to be
contended.

**References**

- CRD status subresource: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource
- Spec and status: https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#object-spec-and-status
- `meta.SetStatusCondition`: https://pkg.go.dev/k8s.io/apimachinery/pkg/api/meta#SetStatusCondition
