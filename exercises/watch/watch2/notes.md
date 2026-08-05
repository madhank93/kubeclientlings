## watch2 — list-then-watch, with no gap and no replay

```go
list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
...
watcher, err := cs.CoreV1().Pods(ns).Watch(ctx, metav1.ListOptions{
	ResourceVersion: list.ResourceVersion, // resume exactly where the list ended
})
```

**Why it works**

- Every `List` response carries a `metadata.resourceVersion` on the **list
  itself**: the exact point in the API server's history that the snapshot
  represents. Starting a watch there means you see every change *after* the
  snapshot and none of the ones already in it. No gap, no duplicates.
- The three `ResourceVersion` values behave completely differently:
  - `""` (unset) — start from **now**, most recent state. You never see the
    existing objects, and you may miss changes between your List and your Watch.
  - `"0"` — start from **any** cached version. The server replays its watch
    cache, so every existing object arrives as a synthetic `Added`. Cheap on the
    server, but you must be prepared to treat "Added" as "here is the world".
  - a concrete version — resume from exactly there.
- This exercise asserts the first `Added` is `new-pod`, which is only true for
  the third option.

**Key detail:** resourceVersions are **opaque**. They are strings that happen
to look like integers today; you must never parse, compare or increment them.
The only valid operations are "pass it back to the server" and "compare for
equality".

A concrete version can also go stale. If your client is slow or disconnected
long enough for the server's history window (its watch cache, ~5 minutes by
default) to move past you, the watch fails with a `410 Gone`
`watch.Error` event. The required recovery is to List again and restart from
the new version — which is precisely the loop `Reflector` implements inside
every informer, and why "just use an informer" is the standard advice.

**References**

- Resource versions: https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions
- Watch semantics: https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes
- `Reflector` (the pattern, implemented): https://pkg.go.dev/k8s.io/client-go/tools/cache#Reflector
