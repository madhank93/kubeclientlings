## pods1 — the namespace argument, not the metadata, routes the request

```go
_, err := cs.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
got, err := cs.CoreV1().Pods(ns).Get(ctx, "hello", metav1.GetOptions{})
```

**Why it works**

- `cs.CoreV1()` picks the API **group/version** (core, `v1`); `.Pods(ns)` picks
  the **resource** and its **namespace**. Together they build the REST path
  `/api/v1/namespaces/<ns>/pods`. The typed client is a thin, generated wrapper
  over exactly that URL.
- `Create` POSTs to the collection URL; `Get` GETs `.../pods/hello`. Both take
  a `context.Context` first — that is your cancellation and deadline, and every
  client-go call has one.
- Create returns the **server's** copy of the object, not yours: it has
  `uid`, `resourceVersion`, `creationTimestamp` and all the defaulted spec
  fields filled in. Getting it back proves it round-tripped.

**Under the hood**

- The generated client is a thin wrapper over `rest.Request`: `Pods(ns)` stores
  the namespace on a `client` struct, and `Create` builds
  `NamespaceIfScoped(ns, len(ns) > 0).Resource("pods").Body(pod)`. Nothing reads
  `pod.Namespace` on the way out.
- Server-side, the request goes through defaulting, validation and admission
  before storage, which is why the returned object differs from yours: `uid`,
  `creationTimestamp`, `resourceVersion`, the default service account,
  `terminationGracePeriodSeconds`, the scheduler name.

**Common mistake**

- Setting `pod.Namespace` and passing `Pods("")`, expecting the object to route
  itself. It does not: the empty argument builds the cluster-wide collection
  URL, and POSTing there fails. The namespace argument routes; the field is
  optional metadata that must not contradict it.

**Key detail:** with `Pods("")` the client builds the cluster-wide path
`/api/v1/pods`, and POSTing there is not valid — you get an error instead of a
pod in the default namespace, which is the surprise most people expect. When
the namespace argument and `pod.Namespace` *both* have values and they
disagree, the server rejects the request with a 400. Setting the namespace on
the object is optional; getting the argument right is not.

`Pods("")` **is** meaningful for reads: `Pods("").List(...)` is `kubectl get
pods --all-namespaces`. Read across all namespaces, write into one.

**See also:** pods2 (filtering the collection URL) · setup2 (where `cs` comes from) ·
dyn1 (the same URL, addressed by hand) · the [pods chapter](../README.md)

**References**

- Typed CoreV1 client: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1
- Object metadata: https://kubernetes.io/docs/concepts/overview/working-with-objects/object-management/
- `metav1.CreateOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#CreateOptions
