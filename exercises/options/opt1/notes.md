## opt1 — building selectors programmatically

```go
selector := labels.SelectorFromSet(labels.Set{"env": "prod", "tier": "web"})

pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
	LabelSelector: selector.String(),
})
```

**Why it works**

- `labels.Set` is just `map[string]string` with methods. `SelectorFromSet` turns
  it into a `labels.Selector` requiring **every** pair — the keys are ANDed, so
  adding `tier: web` narrows `env=prod` from three pods to two.
- `.String()` renders the canonical query form (`env=prod,tier=web`) with keys
  sorted, which is what goes into `ListOptions.LabelSelector`. Going through the
  type rather than `fmt.Sprintf` means malformed keys are caught by the parser
  instead of by the API server.
- A `labels.Selector` can also be evaluated **locally** with
  `selector.Matches(labels.Set(pod.Labels))`. That is how informer-backed
  listers filter without a network call, and why the same type appears in both
  places.

**Under the hood**

- A `labels.Selector` is a slice of `Requirement{key, operator, values}`.
  `String()` renders them sorted so the same set always produces the same query
  string — which matters, because that string is part of the watch cache key on
  the server.
- The same `Requirement` slice powers `Matches()`, so the selector you send to
  the apiserver and the one a lister evaluates in memory are literally the same
  value evaluated by two different backends.

**Common mistake**

- Assembling the query string with `fmt.Sprintf`. It escapes nothing and
  validates nothing, so a value containing a comma or a space becomes a
  different selector than you meant — and one that still parses, so no error
  ever surfaces.

**Key detail:** `SelectorFromSet` is equality-only. For anything richer, build
requirements explicitly:

```go
req, err := labels.NewRequirement("tier", selection.In, []string{"web", "api"})
selector := labels.NewSelector().Add(*req)
```

`labels.NewRequirement` validates the key and values; `labels.Parse("env=prod,tier in (web,api)")`
is the other direction, parsing the string form back into a selector.

Note that `labels.SelectorFromSet` skips validation for speed. Its stricter
sibling `labels.ValidatedSelectorFromValidatedSet` (and plain
`labels.Parse`) will reject an invalid key — use those on anything
user-supplied.

**See also:** opt2 (the other filtering axis) · pods2 (this selector on a real list) ·
inf4 (the same selectors, narrowing an informer) · the
[options chapter](../README.md)

**References**

- `labels` package: https://pkg.go.dev/k8s.io/apimachinery/pkg/labels
- Labels and selectors: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
- `selection` operators: https://pkg.go.dev/k8s.io/apimachinery/pkg/selection
