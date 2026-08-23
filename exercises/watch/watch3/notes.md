## watch3 — a watch that survives dropped connections

```go
list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
...
watcher, err := watchtools.NewRetryWatcherWithContext(ctx, list.ResourceVersion, &cache.ListWatch{
	WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
		return cs.CoreV1().Pods(ns).Watch(ctx, options)
	},
})
```

**Why it works**

- A raw watch is *expected* to end — on a timer, by design. The API server
  computes each watch's lifetime as
  `minRequestTimeout * (rand.Float64() + 1.0)`: a random value between 1× and
  2× the `--min-request-timeout` flag. The randomisation is deliberate, so that
  a thousand clients that connected together don't all reconnect together.
  Load balancers cutting long connections and ordinary network blips do the
  rest. Every one of those closes `ResultChan()`, and naive code reads the
  closed channel and exits.
- `RetryWatcher` wraps your watch function: when the stream dies it reconnects,
  passing the resourceVersion of the **last event it delivered**. Your consumer
  sees one uninterrupted channel and never learns a reconnect happened.
- It takes a `cache.ListWatch` because it needs to *re-invoke* the watch, not
  just consume one. Note this is `WatchFuncWithContext` — the context-aware
  variant, which is what lets `ctx` cancellation propagate into the retry loop.

**Under the hood**

- `RetryWatcher` runs its own loop: it calls your watch function, forwards every
  event while recording the `resourceVersion` of the last one delivered, and on
  channel close reconnects from that version after a short backoff. Consumers
  see one channel and never learn a reconnect happened.
- It classifies errors: a `watch.Error` carrying `Reason: Expired` (410) is
  terminal and closes the result channel, while transient failures are retried.

**Common mistake**

- Passing `"0"` because "it worked with a plain watch". `NewRetryWatcher`
  refuses to construct — deliberately. Neither `""` nor `"0"` identifies a point
  in history, so after a reconnect the watcher could not promise it had neither
  skipped nor replayed events.

**Key detail:** `RetryWatcher` rejects `""` and `"0"` at construction — it
returns an error rather than starting. That refusal is deliberate: neither value
identifies a point in history, so after a reconnect the watcher could not
guarantee it hadn't skipped or replayed events. Forcing a real resourceVersion
forces you into list-then-watch, the only pattern that is actually correct.

What it does **not** do is recover from a `410 Gone`. If the server's history
window passes you, the resourceVersion the watcher is holding becomes invalid
and no amount of retrying helps — you have to List again and construct a new
watcher. Handling that is one of the things an informer's `Reflector` adds on
top, which is the honest reason to prefer informers for anything long-lived.

**See also:** watch2 (the list-then-watch this depends on) · inf1 (the `Reflector`, which
also handles 410) · setup3 (why a watch client wants no `Timeout`) · the
[watch chapter](../README.md)

**References**

- `watchtools.NewRetryWatcher`: https://pkg.go.dev/k8s.io/client-go/tools/watch#NewRetryWatcher
- `cache.ListWatch`: https://pkg.go.dev/k8s.io/client-go/tools/cache#ListWatch
- API concepts — watch: https://kubernetes.io/docs/reference/using-api/api-concepts/
