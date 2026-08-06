## opt2 — field selectors are a different axis from labels

```go
pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
	FieldSelector: "metadata.name=web-2", // FieldSelector, not LabelSelector
})
```

**Why it works**

- `LabelSelector` and `FieldSelector` are two independent query parameters.
  Labels query `metadata.labels`; field selectors query the object's **own
  fields**. `metadata.name` is a field, and no pod carries it as a label — so
  putting it in `LabelSelector` matches nothing and, crucially, does **not**
  error. You get an empty list, which is the hardest kind of bug to spot.
- Field selectors only work on fields the API server has explicitly indexed for
  that resource. For Pods that is `metadata.name`, `metadata.namespace`,
  `spec.nodeName`, `spec.schedulerName`, `spec.restartPolicy`,
  `spec.serviceAccountName`, `status.phase`, `status.podIP`,
  `status.nominatedNodeName`. Anything else — `spec.containers[0].image`, say —
  is rejected outright with a 400.
- Supported operators are only `=`, `==` and `!=`; comma-separated terms are
  ANDed. No set operations, no existence tests.

**Key detail:** the two behave differently on a typo, and that asymmetry is
worth internalising. An unsupported *field* selector is a loud 400. An
unmatched *label* selector is a quiet empty list. When a list comes back
empty and you expected rows, check that you filtered on the axis you think you
did.

The single most useful field selector in real controllers is
`spec.nodeName=<node>` — it's how the kubelet watches only its own pods instead
of the whole cluster, and how you'd write anything node-scoped.

Building them safely: `fields.OneTermEqualSelector("metadata.name", "web-2").String()`,
or `fields.AndSelectors(...)` for several terms.

**References**

- Field selectors: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
- `fields` package: https://pkg.go.dev/k8s.io/apimachinery/pkg/fields
- `metav1.ListOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
