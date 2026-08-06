## setup2 — from rest.Config to a working clientset

```go
clientset, err := kubernetes.NewForConfig(config) // the loaded config, not &rest.Config{}
if err != nil {
	fmt.Printf("❌ could not create clientset: %v\n", err)
	os.Exit(1)
}
nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
```

**Why it works**

- `kubernetes.NewForConfig` copies the config, applies defaults (content type,
  negotiated serializer, rate limiter) and builds one typed client per
  group/version. `clientset.CoreV1()`, `AppsV1()`, `BatchV1()` are all just
  accessors on that one struct — they share the underlying HTTP transport and
  connection pool, which is why you build the clientset **once** and pass it
  around.
- `Nodes()` takes no namespace argument because Node is a cluster-scoped
  resource. Compare with `Pods(ns)` in the next topic: the method signature is
  how client-go encodes scope.
- `List` returns a `*corev1.NodeList`; the objects live in `.Items`, and the
  list itself carries a `ResourceVersion` you will use later for watches.

**Key detail:** the two bugs here are the ones that actually happen in real
code.

1. `&rest.Config{}` compiles fine and even constructs a clientset — an empty
   config just means "no host, no credentials", so the failure lands on the
   *first request* as a dial error against an empty URL, far from the cause.
2. `clientset, _ := ...` swallows the error. Every client-go constructor
   returns an error for a reason (bad TLS material, unparseable host, a broken
   exec-credential plugin), and discarding it converts a clear startup failure
   into a nil-ish client that panics or misbehaves later.

**References**

- `kubernetes.NewForConfig`: https://pkg.go.dev/k8s.io/client-go/kubernetes#NewForConfig
- Clientset API: https://pkg.go.dev/k8s.io/client-go/kubernetes#Clientset
- client-go usage guide: https://github.com/kubernetes/client-go/blob/master/INSTALL.md
