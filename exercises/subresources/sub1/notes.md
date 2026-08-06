## sub1 — /scale is a separate endpoint with its own object

```go
scale, err := cs.AppsV1().Deployments(ns).GetScale(ctx, "web", metav1.GetOptions{})
scale.Spec.Replicas = 3 // DESIRED count — Status.Replicas is observed, read-only
_, err = cs.AppsV1().Deployments(ns).UpdateScale(ctx, "web", scale, metav1.UpdateOptions{})
```

**Why it works**

- `/scale` is a **subresource**: a different URL
  (`.../deployments/web/scale`) exposing a different type,
  `autoscaling/v1.Scale`. It has just `spec.replicas`, `status.replicas` and
  `status.selector` — a tiny, uniform view of "how many of this thing".
- `Spec.Replicas` is desired; `Status.Replicas` is observed. The server ignores
  status on write, so setting it changes nothing at all and the deployment stays
  at 1. Same spec/status split as everywhere else in the API — the exercise just
  makes it visible.
- Uniformity is the payoff. Deployments, StatefulSets, ReplicaSets and any CRD
  that enables `subresources.scale` all expose the *same* `Scale` type. That's
  how one HPA controller scales resources it has no compile-time knowledge of,
  and how `kubectl scale` works on your CRD.

**Key detail:** RBAC is per-subresource. You can grant
`update` on `deployments/scale` without granting `update` on `deployments` — the
holder can resize but cannot change the image, the selector or anything else.
That's exactly the permission an autoscaler should have, and it is impossible to
express with a whole-object Update.

The other reason to use `/scale`: writing `spec.replicas` via a full
`Deployments().Update()` sends the *entire* object, so you can clobber a
concurrent change to an unrelated field. `UpdateScale` touches one number.

Beware the HPA interaction, though: if an HPA manages a Deployment, anything you
write to `spec.replicas` (through either path) is transient — the HPA will move
it back on its next tick.

**References**

- `autoscaling/v1.Scale`: https://pkg.go.dev/k8s.io/api/autoscaling/v1#Scale
- Subresources in the API: https://kubernetes.io/docs/reference/using-api/api-concepts/
- CRD scale subresource: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource
