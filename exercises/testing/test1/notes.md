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

**Under the hood**

- `fake.Clientset` embeds `testing.Fake` and installs a default
  `ObjectReaction` bound to a `testing.ObjectTracker`. Every generated method
  builds the same `Action` value the real client would have sent as a request,
  and hands it to the reactor chain instead of a transport.
- The tracker stores objects in `map[schema.GroupVersionResource][]runtime.Object`
  and applies real Get/Create/Update/Delete semantics, including
  `AlreadyExists` and `NotFound`.

**Common mistake**

- Accepting `*kubernetes.Clientset` in your own function signatures. The fake
  cannot be substituted for a concrete struct, so the code becomes untestable
  without a cluster — take `kubernetes.Interface` instead.

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

Note on the two constructors: `NewSimpleClientset` is **not** deprecated (no
`Deprecated:` marker in client-go), but `NewClientset` is the one to reach for.
Its doc comment states the difference:

> Compared to NewSimpleClientset, the Clientset returned here supports field
> tracking and thus server-side apply.

So if the code under test uses `Apply` (the `ssa` topic), you need
`NewClientset` — under `NewSimpleClientset` the apply path has no
`managedFields` to track. That same comment warns SSA support for **CRDs** is
still missing, so an apply against a custom resource is not faithfully faked by
either.

**See also:** test2 (making the fake fail on demand) · test3 (asserting on what it
recorded) · setup2 (the real clientset behind the same interface) · the
[testing chapter](../README.md)

**References**

- `kubernetes/fake`: https://pkg.go.dev/k8s.io/client-go/kubernetes/fake
- `testing.ObjectTracker`: https://pkg.go.dev/k8s.io/client-go/testing#ObjectTracker
- envtest: https://book.kubebuilder.io/reference/envtest.html
