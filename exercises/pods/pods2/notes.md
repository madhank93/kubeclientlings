## pods2 — let the server do the filtering

```go
webPods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
	LabelSelector: "app=web",
})
```

**Why it works**

- `ListOptions.LabelSelector` is sent as the `?labelSelector=` query parameter.
  The **API server** evaluates it and only matching objects cross the wire —
  the same string syntax `kubectl get -l` takes: `app=web`,
  `app!=db`, `app in (web,api)`, `tier` (key exists), `!tier` (key absent),
  comma-separated for AND.
- Filtering server-side matters at scale: listing 50,000 pods to keep 3 costs
  the API server memory, the network bandwidth, and your process a large
  deserialisation. Never `List` everything and filter in Go if a selector can
  do it.
- Prefer building the string with `labels.Set{"app": "web"}.AsSelector().String()`
  or `labels.SelectorFromSet(...)` in real code — it escapes values properly
  instead of relying on `fmt.Sprintf`.

**Under the hood**

- `ListOptions` is encoded by `VersionedParams` into query parameters, so
  `LabelSelector` arrives as `?labelSelector=app%3Dweb`. The apiserver parses it
  with the same `labels` package you have locally, then evaluates it against the
  watch cache or etcd before serialising anything.
- Because the filtering happens before serialisation, a selective list costs the
  server almost nothing extra — the expensive part of a `List` is
  materialising and encoding objects, not finding them.

**Common mistake**

- Listing everything and filtering in Go, usually because the selector "didn't
  work". A selector that matches nothing returns an empty list, never an error,
  so a typo looks exactly like an empty namespace — and the fix people reach for
  is the one that hurts the cluster.

**Key detail:** `FieldSelector` is the other half of this and is **not**
interchangeable. Field selectors work on object fields (`metadata.name`,
`status.phase`, `spec.nodeName`) but only on fields the server has explicitly
indexed for that resource — `spec.containers[0].image` is not one of them.
Label selectors work on anything you labelled. When you need to select on
something arbitrary, put a label on it.

Also note `List` is a point-in-time snapshot with a `ResourceVersion` on the
list itself. That version is what you hand to a `Watch` to resume without a gap
— which is exactly what informers do internally.

**See also:** opt1 (building selectors through the `labels` package) · opt2 (the other
filtering axis) · opt5 (bounding a large list) · the
[pods chapter](../README.md)

**References**

- Labels and selectors: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
- `metav1.ListOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
- `labels` helpers: https://pkg.go.dev/k8s.io/apimachinery/pkg/labels
