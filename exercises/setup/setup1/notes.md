## setup1 — building a rest.Config the way kubectl does

```go
rules := clientcmd.NewDefaultClientConfigLoadingRules()
overrides := &clientcmd.ConfigOverrides{CurrentContext: "kind-kubeclientlings"}
config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
```

**Why it works**

- `NewDefaultClientConfigLoadingRules()` encodes kubectl's own search order:
  `$KUBECONFIG` (colon-separated, merged left-to-right) first, then
  `~/.kube/config`. Nothing is read yet — the rules are a *recipe*.
- `ConfigOverrides` is the in-process equivalent of kubectl's flags. Setting
  `CurrentContext` is exactly `kubectl --context=kind-kubeclientlings`, so the
  exercise talks to the kind cluster no matter what your shell's current
  context is.
- `NewNonInteractiveDeferredLoadingClientConfig` is **deferred**: the files are
  only opened when you call `ClientConfig()`. "Non-interactive" means it will
  never prompt on a terminal for a missing auth field — it errors instead,
  which is what you want in a controller.
- `ClientConfig()` collapses everything (cluster address, CA bundle, client
  cert / token / exec plugin) into one `*rest.Config` — the credential bundle
  every client-go constructor takes from here on.

**Under the hood**

- `DeferredLoadingClientConfig` holds the rules and overrides and does nothing
  until asked. `ClientConfig()` then runs `Load()` over each file in the search
  path and **merges** them: contexts, clusters and users are unioned, and the
  *first* file to define a given name wins. That is why prepending a file to
  `$KUBECONFIG` shadows entries later in the list.
- The merged result is collapsed into `*rest.Config` by resolving the current
  context to its cluster and user, then translating the auth stanza into
  transport material — `CertFile`/`KeyData`, `BearerToken`, or an `ExecProvider`
  that shells out to a credential plugin on first use.

**Common mistake**

- Assuming a loaded config means a *usable* one. Loading succeeds against a
  kubeconfig whose current-context points at a cluster entry that no longer
  exists; you get a `*rest.Config` with an empty `Host` and the first request
  dies dialing nothing. Assert on `config.Host` at startup rather than
  discovering it four layers down.

**Key detail:** `clientcmd.BuildConfigFromFlags("", path)` is the shortcut you
see in older examples, and it is *not* the same thing — it takes one explicit
path and ignores `$KUBECONFIG` entirely. Reach for it only when a user has
literally passed you a `--kubeconfig` flag. In-cluster code wants
`rest.InClusterConfig()` instead, and `BuildConfigFromFlags("", "")` silently
falls back to that.

`config.Host` being empty is the tell-tale of a config that "loaded" but
resolved to nothing — worth asserting on, because the failure otherwise
surfaces much later as a confusing dial error.

**See also:** setup2 (turning this config into a clientset) · setup3 (the fields worth
setting before you do) · the [setup chapter](../README.md)

**References**

- clientcmd package: https://pkg.go.dev/k8s.io/client-go/tools/clientcmd
- `rest.Config` fields: https://pkg.go.dev/k8s.io/client-go/rest#Config
- Kubeconfig merge rules: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/
