## svc1 — a Service selector nobody validates

```go
Selector: map[string]string{"app": "web"}, // must match the pods' labels
```

**Why it works**

- `Service.spec.selector` is a plain `map[string]string` — equality-only, ANDed
  across keys. There is no `matchExpressions` here; Services predate the richer
  `LabelSelector` type and never got it.
- The endpoints controller runs that selector against pods in the namespace and
  writes the matching, *ready* pod IPs into EndpointSlices. kube-proxy programs
  the dataplane from those slices. Get the selector wrong and the whole chain
  produces nothing.
- `labels.Set(svc.Spec.Selector).String()` converts the map into the
  `"app=web"` query string form, which is how the exercise re-runs the
  Service's own selector as a `List` and proves it selects something.

**Key detail:** nothing validates this at admission. A Service whose selector
matches zero pods is a completely legal object — it gets a ClusterIP, DNS
resolves, and connections just fail to connect. That silence is what makes it
one of the most common production misconfigurations. In your own code, assert
that the selector selects something (exactly as this exercise does) rather than
trusting the create to have caught it.

Two adjacent facts worth knowing:

- A Service with **no** selector at all is also legal and deliberate — it means
  "I'll manage the EndpointSlices myself", the standard way to point a Service
  at an external address.
- The selector matches pods, not Deployments. Two Deployments whose pods share
  a label are both behind the Service — which is the mechanism behind blue/green
  and canary routing.

**References**

- Services: https://kubernetes.io/docs/concepts/services-networking/service/
- `ServiceSpec`: https://pkg.go.dev/k8s.io/api/core/v1#ServiceSpec
- `labels.Set`: https://pkg.go.dev/k8s.io/apimachinery/pkg/labels#Set
