## watch1 — event types are constants, not strings you invent

```go
switch event.Type {
case watch.Added:
	sawAdded = true
case watch.Deleted:
	sawDeleted = true
}
```

**Why it works**

- `Watch` opens a long-lived, chunked HTTP response and hands you a
  `watch.Interface`. `ResultChan()` yields `watch.Event{Type, Object}` values
  until the stream ends; `Stop()` closes it — always `defer` that, or you leak
  a connection and a goroutine.
- `watch.EventType` is a **defined string type**, so `case "DELETE"` compiles
  without complaint. It just never matches: the real constants are
  `watch.Added`, `watch.Modified`, `watch.Deleted`, `watch.Bookmark` and
  `watch.Error`, whose underlying values are `"ADDED"`, `"MODIFIED"`,
  `"DELETED"`, `"BOOKMARK"`, `"ERROR"`. Use the constants and the compiler
  catches your typos.
- `event.Object` is a `runtime.Object`, so it needs a type assertion
  (`pod, ok := event.Object.(*corev1.Pod)`) before you can read fields. Always
  use the two-value form — on a `watch.Error` event the object is a
  `*metav1.Status`, and a single-value assertion would panic.

**Under the hood**

- A watch is one HTTP response with `Transfer-Encoding: chunked`; each chunk is
  a serialised `metav1.WatchEvent`. `StreamWatcher` decodes them on a goroutine
  and pushes onto the channel `ResultChan()` hands you, so `Stop()` is what ends
  that goroutine and closes the body.
- The channel is unbuffered past a small window: a slow consumer applies
  backpressure all the way to the apiserver, and a consumer that stops reading
  entirely is eventually disconnected.

**Common mistake**

- Asserting `event.Object.(*corev1.Pod)` with the single-value form. It works
  for every normal event and panics the first time a `watch.Error` arrives
  carrying a `*metav1.Status` — which is precisely the moment you most needed
  the program alive.

**Key detail:** two event types exist that aren't object changes.
`watch.Error` carries a `*metav1.Status` — most often "resourceVersion too
old" (HTTP 410 Gone), meaning the server's history window has passed you by and
you must re-List. `watch.Bookmark` carries no useful object but does carry an
updated `resourceVersion`, letting a client checkpoint its position on a quiet
watch so it can resume without a full relist.

Also: the deletion here passes `GracePeriodSeconds: 0` on purpose. The default
30-second graceful shutdown would delay the `Deleted` event past the window the
exercise watches.

Raw `Watch` is the primitive under everything else. Production code almost
never uses it directly — it uses an informer (see the `informers` topic), which
adds relisting, reconnection and a local cache on top.

**See also:** watch2 (starting the stream at the right point) · watch3 (surviving the
stream ending) · inf1 (the machinery that wraps all of this) · the
[watch chapter](../README.md)

**References**

- `watch` package: https://pkg.go.dev/k8s.io/apimachinery/pkg/watch
- Efficient detection of changes: https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes
- Watch bookmarks: https://kubernetes.io/docs/reference/using-api/api-concepts/#watch-bookmarks
