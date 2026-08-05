## test1 — the fake clientset and its object tracker

```go
cs := fake.NewClientset(
	&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "seeded", Namespace: "demo"}},
)

got, err := cs.CoreV1().Pods("demo").Get(ctx, "seeded", metav1.GetOptions{})
```

**Why it works**

- `fake.Clientset` implements `kubernetes.Interface` — the *same* interface the
  real clientset implements. So any function you write that takes
  `kubernetes.Interface` (rather than `*kubernetes.Clientset`) is testable with
  no cluster, no network and no flakes. That's the practical reason to accept
  the interface in your own signatures.
- Behind it sits an `ObjectTracker`: an in-memory store keyed by
  **GVR + namespace + name**. `NewClientset(objs...)` seeds it; every
  Get/List/Create/Update/Delete afterwards operates on it with real semantics —
  Create on an existing name returns `AlreadyExists`, Get on a missing one
  returns `NotFound`.
- Namespace is part of the key. Seeding into `demo` and querying `default` is
  an ordinary `NotFound`, not a fallback or a wildcard — same rule as the real
  API.

**Key detail:** the fake is a good imitation, not a real API server, and the
gaps are where tests give false confidence:

- **No validation, no defaulting, no admission.** You can create a Pod with an
  empty spec and a Deployment whose selector doesn't match its template. Both
  would be rejected by a real cluster.
- **No controllers.** Create a Deployment and no ReplicaSet or Pods appear;
  delete an owner and nothing is garbage collected. `ownerReferences` are inert
  strings here.
- **`resourceVersion` is not maintained** the way the real server does it, so
  optimistic-concurrency behaviour is not faithfully reproduced.
- Objects passed in are **not** deep-copied on the way in, so mutating a struct
  you seeded can change what the tracker holds.

For anything that depends on real API-server behaviour, use `envtest`
(controller-runtime), which runs an actual `kube-apiserver` and `etcd` binary.
The fake is for unit tests of *your* logic; envtest is for integration tests of
the interaction.

Note `fake.NewClientset` is the current constructor — the older
`fake.NewSimpleClientset` is deprecated.

**References**

- `kubernetes/fake`: https://pkg.go.dev/k8s.io/client-go/kubernetes/fake
- `testing.ObjectTracker`: https://pkg.go.dev/k8s.io/client-go/testing#ObjectTracker
- envtest: https://book.kubebuilder.io/reference/envtest.html
