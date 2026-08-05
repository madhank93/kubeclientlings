## inf2 — registering a handler starts nothing

```go
_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	AddFunc: func(obj any) { ... },
})
...
factory.Start(ctx.Done())
if !cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced) {
	exkit.Failf("cache never synced")
}
```

**Why it works**

- `AddEventHandler` appends a listener to a `sharedProcessor`. It allocates a
  goroutine-fed ring buffer and returns a registration handle — and that is
  *all*. No watch, no HTTP, no goroutines running the informer itself.
- `factory.Start(stopCh)` is what launches each informer's `Run` loop. Until
  then nothing lists, nothing watches, and no handler can possibly fire. The
  program compiles, runs, and blocks forever on a channel nobody will send to —
  the failure has no error message at all, which is what makes it the classic
  informer bug.
- Handlers must be registered **before** `Start`, or you may miss the initial
  batch. Ordering matters here in a way it usually doesn't.

**Key detail:** during the initial sync, every pre-existing object is delivered
to `AddFunc` — the informer doesn't distinguish "this already existed" from
"this was just created". Your handler must therefore be **idempotent**. This is
the deep reason controllers are *level-triggered*: handlers do not act on the
event, they enqueue a key, and the reconcile loop looks at current state. Then
"created" versus "replayed at startup" stops mattering.

Two more things the handler contract requires:

- `obj any` needs a type assertion, and on delete you may receive a
  `cache.DeletedFinalStateUnknown` tombstone instead of the object — that's the
  informer telling you it missed the delete and resynced. Handle it:
  `if tomb, ok := obj.(cache.DeletedFinalStateUnknown); ok { obj = tomb.Obj }`.
- `UpdateFunc(old, new any)` fires on **every** update, including resyncs where
  nothing changed. Compare `ResourceVersion` and return early if they match.

Handlers run on the informer's own goroutine, serialised per listener. Blocking
in one blocks that informer's delivery to that listener — never do I/O in a
handler. Enqueue and return; the `workqueue` topic is the other half of this.

**References**

- `cache.ResourceEventHandler`: https://pkg.go.dev/k8s.io/client-go/tools/cache#ResourceEventHandler
- `SharedInformer`: https://pkg.go.dev/k8s.io/client-go/tools/cache#SharedInformer
- sample-controller: https://github.com/kubernetes/sample-controller
