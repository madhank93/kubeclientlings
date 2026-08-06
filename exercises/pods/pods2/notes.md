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

**Key detail:** `FieldSelector` is the other half of this and is **not**
interchangeable. Field selectors work on object fields (`metadata.name`,
`status.phase`, `spec.nodeName`) but only on fields the server has explicitly
indexed for that resource — `spec.containers[0].image` is not one of them.
Label selectors work on anything you labelled. When you need to select on
something arbitrary, put a label on it.

Also note `List` is a point-in-time snapshot with a `ResourceVersion` on the
list itself. That version is what you hand to a `Watch` to resume without a gap
— which is exactly what informers do internally.

**References**

- Labels and selectors: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
- `metav1.ListOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
- `labels` helpers: https://pkg.go.dev/k8s.io/apimachinery/pkg/labels
