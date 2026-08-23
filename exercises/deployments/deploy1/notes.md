## deploy1 — the selector/template-label contract

```go
labels := map[string]string{"app": "web"} // one map, used in both places
Selector: &metav1.LabelSelector{MatchLabels: labels},
Template: corev1.PodTemplateSpec{
	ObjectMeta: metav1.ObjectMeta{Labels: labels},
	...
}
```

**Why it works**

- A Deployment doesn't hold references to its pods. It finds them by **label
  query**: `spec.selector` is run against pods in the namespace, and whatever
  matches is considered owned. `spec.template.metadata.labels` is what the pods
  it stamps out will carry.
- If the two disagree, the Deployment creates pods it cannot see, decides it
  still has zero replicas, and creates more — forever. Rather than let you build
  that, the API server validates the relationship at admission and rejects the
  object with a 422.
- Sharing one `labels` variable between the selector and the template makes the
  invariant structural instead of something you have to remember.

**Under the hood**

- Validation runs `unversionedvalidation.ValidateLabelSelector` and then checks
  the template's labels against it with `Matches` — the same evaluation a `List`
  would do, executed at admission time.
- Once accepted, the Deployment controller hashes `spec.template`, appends
  `pod-template-hash=<hash>` to the ReplicaSet's selector and template, and lets
  the ReplicaSet controller own the pods. That extra label is what keeps two
  generations of pods apart under one Deployment selector.

**Common mistake**

- Building the selector and the template labels from two separate literals that
  happen to agree today. Admission catches a mismatch on create, but a later
  edit to the template labels alone passes validation only if it still matches —
  and the selector cannot be edited to follow, because it is immutable.

**Key detail:** `spec.selector` is **immutable** after creation on Deployments,
StatefulSets and DaemonSets (apps/v1). You cannot fix a bad selector with an
Update — you delete the object and recreate it. Get it right the first time.

`MatchLabels` is the simple equality form. The richer `MatchExpressions`
(`In`, `NotIn`, `Exists`, `DoesNotExist`) is also available, and
`metav1.LabelSelectorAsSelector(sel)` converts either form into the string you
would pass to `ListOptions.LabelSelector`.

The ownership chain is worth having in your head: Deployment → (selector) →
ReplicaSet → (selector) → Pods, with `metadata.ownerReferences` recorded on the
children for garbage collection.

**See also:** deploy2 (scaling the object you just created) · deploy4 (knowing when its
rollout finished) · opt4 (the ownerReferences behind the chain) · svc1 (the
other selector that finds these pods) · the
[deployments chapter](../README.md)

**References**

- Deployments: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
- `metav1.LabelSelector`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#LabelSelector
- Owners and dependents: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
