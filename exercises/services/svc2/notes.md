## svc2 — finding EndpointSlices by their well-known label

```go
slices, err := cs.DiscoveryV1().EndpointSlices(ns).List(ctx, metav1.ListOptions{
	LabelSelector: discoveryv1.LabelServiceName + "=web", // "kubernetes.io/service-name"
})
```

**Why it works**

- EndpointSlices are **not** named after their Service — the controller
  generates names like `web-x7k2p`, and there can be several per Service. The
  control plane packs at most **100** endpoints per slice by default
  (`--max-endpoints-per-slice` on kube-controller-manager, raisable to 1000,
  which is also the API's hard cap), and slices are split further by address
  type and port set. So there is nothing to `Get`; you `List` and filter.
- The link back to the Service is the label `kubernetes.io/service-name`,
  exported as the constant `discoveryv1.LabelServiceName`. Using the constant
  rather than a hand-typed string is the whole lesson: a wrong label selector
  is not an error, it's an empty list — a silent failure that looks exactly like
  "no endpoints yet" and will happily spin until your timeout.
- `ep.Conditions.Ready` is a `*bool`, so it needs the nil check before the
  dereference: `nil` means "unknown", which is not the same as `false`.

**Key detail:** the old `v1.Endpoints` API (a single object named exactly after
the Service) is **deprecated as of Kubernetes 1.33** — the API server now emits
warnings when you read or write Endpoints. It doesn't scale (every pod change
rewrites one object that every node is watching) and it can't carry topology
hints or dual-stack addresses. New code should read `discovery.k8s.io/v1`
EndpointSlices.

Note this deprecation is **not** listed in the Deprecated API Migration Guide —
that page tracks *removals*, and Endpoints has not been removed. The
`[deprecated]` feature-state badge is on the EndpointSlice concept page.

Endpoints only appear once pods are **ready**, so a wait loop here is not
optional. `Conditions` also carries `Serving` and `Terminating`, which let you
distinguish "still draining connections" from "gone" — the basis of graceful
shutdown.

**References**

- EndpointSlices: https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/
- `discoveryv1` constants: https://pkg.go.dev/k8s.io/api/discovery/v1#pkg-constants
- Endpoints deprecation (v1.33 announcement): https://kubernetes.io/blog/2025/04/24/endpoints-deprecation/
