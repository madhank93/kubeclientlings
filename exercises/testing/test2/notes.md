## test2 — reactors: testing the sad path

```go
cs.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
	return true, nil, apierrors.NewInternalError(fmt.Errorf("etcd unavailable"))
})
```

**Why it works**

- Every call on the fake clientset walks a chain of **reactors** before reaching
  the default tracker. A reactor is `func(Action) (handled bool, ret
  runtime.Object, err error)`.
  - `handled == true` — this reactor answers; the chain stops and the tracker is
    never consulted.
  - `handled == false` — fall through to the next reactor, and eventually the
    tracker.
- Registration is `(verb, resource)` and it must match the call you want to
  intercept. `"update"` does not fire on a Create — the reactor is simply never
  invoked, the tracker handles the create normally, and `err` is `nil`. Nothing
  warns you that your reactor was dead code.
- `PrependReactor` puts yours at the **front**, ahead of the default tracker
  reactor that `NewClientset` installs. `AddReactor` appends, which for most
  purposes means "after the tracker already answered" — usually not what you
  want.
- Both arguments accept `"*"` as a wildcard: `PrependReactor("*", "*", ...)`
  breaks everything.

**Under the hood**

- `Fake.Invokes(action, defaultReturn)` walks `ReactionChain` in order, calling
  each reactor whose `(verb, resource)` matches until one returns
  `handled=true`. The tracker's own reactor is just the last link, installed by
  the constructor.
- Because the chain is ordered, `PrependReactor` is what puts yours ahead of the
  tracker; `AddReactor` appends behind it, where the tracker has usually already
  answered.

**Common mistake**

- Registering a reactor whose verb does not match the call — `"update"` for a
  Create, or the singular resource name. Nothing warns you; the reactor is dead
  code, the tracker answers normally, and the test passes for the wrong reason.

**Key detail:** this is the only practical way to test the paths that matter
most — `IsConflict` retry loops, `IsNotFound` handling, `IsTooManyRequests`
backoff, and "the API server went away mid-reconcile". A live cluster will not
produce those on demand.

Use the real constructors (`apierrors.NewInternalError`, `NewConflict`,
`NewNotFound`) rather than `errors.New`, so the `apierrors.Is*` checks in your
code under test actually match.

The `Action` argument carries the details, so a reactor can be selective:

```go
cs.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
	pod := action.(k8stesting.CreateAction).GetObject().(*corev1.Pod)
	if pod.Name != "doomed" {
		return false, nil, nil // not ours — fall through to the tracker
	}
	return true, nil, apierrors.NewInternalError(fmt.Errorf("etcd unavailable"))
})
```

Closing over a counter is the standard way to fail the first N attempts and then
succeed — exactly what a retry test needs.

**See also:** test1 (the tracker being intercepted) · test3 (checking what actually ran) ·
pods5 (the `apierrors` predicates these injected errors must satisfy) · pods3
(the conflict-retry path worth testing this way) · the
[testing chapter](../README.md)

**References**

- `client-go/testing`: https://pkg.go.dev/k8s.io/client-go/testing
- `apierrors` constructors: https://pkg.go.dev/k8s.io/apimachinery/pkg/api/errors
- `Fake.PrependReactor`: https://pkg.go.dev/k8s.io/client-go/testing#Fake.PrependReactor
