## op2 — Owns() follows ownerRefs back to the parent

```go
if err := controllerutil.SetControllerReference(&parent, child, r.scheme); err != nil {
	return ctrl.Result{}, err
}
if err := r.Create(ctx, child); err != nil && !apierrors.IsAlreadyExists(err) {
	return ctrl.Result{}, err
}
```

with

```go
ctrl.NewControllerManagedBy(mgr).
	For(&corev1.ConfigMap{}).   // reconcile these — request = this object
	Owns(&corev1.Secret{}).     // watch these — request = their OWNER
	Complete(r)
```

**Why it works**

- `For(X)` says "reconcile X"; each event on an X enqueues that X's own key.
- `Owns(Y)` says "also watch Y, but map each event back to the **owner**". The
  mapping function reads the child's *controller* ownerReference, checks the
  Kind matches the `For` type, and enqueues `owner.Namespace/owner.Name`.
- So the ownerRef isn't decoration — it is the entire routing table. A child
  created without one produces events that map to nothing, and deleting it is
  silent. The controller only notices on its next reconcile of the parent, which
  may be never.
- `SetControllerReference` fills in APIVersion, Kind, Name, UID **and**
  `Controller: true`. It needs the `*runtime.Scheme` to look the GVK up from the
  Go type, which is why the reconciler carries `mgr.GetScheme()`.

**Key detail:** `SetControllerReference` and `SetOwnerReference` are different.
An object may have many owner references but at most **one** with
`Controller: true`; `SetControllerReference` errors if a different controller
already claims it. That single controller ref is what `metav1.GetControllerOf`
returns and what `Owns()` follows. Plain `SetOwnerReference` gives you garbage
collection without the routing.

You get GC for free either way: delete the parent and the child is collected
(opt4). Here the interesting direction is the other one — delete the *child* and
the controller resurrects it, which is exactly how a Deployment defends its
ReplicaSet.

Ownership is namespace-local (opt4) and cannot cross namespaces or from
cluster-scoped to namespaced. When you genuinely need cross-namespace or
non-ownership relationships, `Watches()` with a custom
`handler.EnqueueRequestsFromMapFunc` is the escape hatch — you write the mapping
yourself.

Note the `AlreadyExists` tolerance in the create: reconciles are re-entrant, so
"someone (probably an earlier me) already made it" is a normal outcome, not a
failure.

**References**

- Builder & `Owns`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder
- `controllerutil.SetControllerReference`: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil#SetControllerReference
- Owners and dependents: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
