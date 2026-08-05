## opt4 — ownerReferences are keyed by UID, not name

```go
OwnerReferences: []metav1.OwnerReference{{
	APIVersion: "v1",
	Kind:       "ConfigMap",
	Name:       owner.Name,
	UID:        owner.UID, // the server-assigned identity — required
}},
```

**Why it works**

- The garbage collector walks `metadata.ownerReferences` on every object. When
  an owner disappears, everything referencing it is deleted — that is the whole
  mechanism behind "delete the Deployment and the ReplicaSets and Pods go too".
- All four fields identify the owner: `APIVersion` + `Kind` say *what type*,
  `Name` says which one, and `UID` pins the exact **instance**. The UID is
  required because names are reusable: delete `owner` and create a new
  ConfigMap with the same name and you have a different object. Without the
  UID the GC could not tell "my owner still exists" from "my owner was replaced",
  and children would survive deletions or be deleted by strangers.
- You only get a UID from the server, which is why the code creates the owner
  first and reads `owner.UID` off the returned object.

**Key detail:** ownerReferences are **namespace-local**. A namespaced object
cannot be owned by an object in another namespace, and a cluster-scoped object
cannot be owned by a namespaced one. Violating that doesn't error at admission —
the GC just treats the reference as dangling and deletes the child. Debugging
that is unpleasant, so it's worth knowing up front.

`Controller: true` marks the *one* owner that actively manages the object (at
most one per object; it's what `metav1.GetControllerOf` returns, and what
controller-runtime's `Owns()` follows back to a reconcile key).
`BlockOwnerDeletion: true` makes foreground deletion of the owner wait for this
child to go first.

In practice you rarely build this struct by hand — `controllerutil.SetOwnerReference`
and `SetControllerReference` do it, including resolving the GVK from the scheme.

**References**

- Owners and dependents: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
- Garbage collection: https://kubernetes.io/docs/concepts/architecture/garbage-collection/
- `metav1.OwnerReference`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#OwnerReference
