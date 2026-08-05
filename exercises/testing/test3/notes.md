## test3 — Actions() is your audit log

```go
actions := cs.Actions()
actions[0].Matches("create", "configmaps") // (verb, resource) — the call you made

create := actions[0].(k8stesting.CreateAction)
obj := create.GetObject().(*corev1.ConfigMap)
```

**Why it works**

- The fake clientset appends an `Action` for **every** call it receives, in
  order. `Actions()` returns that slice, which turns "what did my controller
  actually do?" into an ordinary assertion.
- `Matches(verb, resource)` compares the recorded verb (`get`, `list`, `watch`,
  `create`, `update`, `patch`, `delete`, `delete-collection`) and the plural
  resource name. Asserting `"update"` against a create is just `false` — the
  usual quiet miss.
- `Action` is an interface; the concrete types carry the payload:
  - `CreateAction` / `UpdateAction` → `GetObject()` — the object that was sent.
  - `PatchAction` → `GetPatch()`, `GetPatchType()`.
  - `GetAction` / `DeleteAction` → `GetName()`.
  - all of them → `GetNamespace()`, `GetSubresource()`.
- Asserting on the payload is the point. "It didn't error" is a weak test;
  "it created a ConfigMap named `audited` in `demo` with `data.k=v`" is the one
  that catches a regression.

**Key detail:** `Actions()` records **reads too**, and that trips people up.
A test that asserts `len(actions) == 1` after a reconcile that did one Get and
one Create fails, and the count is fragile as the code evolves. Filter instead
of counting:

```go
for _, a := range cs.Actions() {
	if a.Matches("create", "configmaps") { ... }
}
```

Assert on the writes you care about, ignore the reads.

`cs.ClearActions()` resets the log between phases of a table test.

The other reason this matters: for **subresource** writes the verb alone is
ambiguous — an `UpdateStatus` records as `update` with
`GetSubresource() == "status"`. If your controller writes both spec and status,
that field is how you tell the two apart.

**References**

- `testing.Action` and friends: https://pkg.go.dev/k8s.io/client-go/testing#Action
- `Fake.Actions`: https://pkg.go.dev/k8s.io/client-go/testing#Fake.Actions
- sample-controller tests: https://github.com/kubernetes/sample-controller/blob/master/controller_test.go
