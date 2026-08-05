## watch4 — an EventRecorder needs a sink

```go
broadcaster := record.NewBroadcaster()
broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
	Interface: cs.CoreV1().Events(ns),
})
defer broadcaster.Shutdown()

recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "kubeclientlings"})
recorder.Event(pod, corev1.EventTypeNormal, "Exercised", "kubeclientlings was here")
```

**Why it works**

The pipeline has three pieces and the middle one is easy to forget:

- **Broadcaster** — an in-memory fan-out. `NewRecorder` gives you a producer;
  `StartRecordingToSink` / `StartLogging` attach consumers.
- **Sink** — where events actually go. `EventSinkImpl{Interface: cs.CoreV1().Events(ns)}`
  writes them to the API server. Without it the broadcaster has *no consumers*,
  so `recorder.Event(...)` succeeds, the event goes into the fan-out, and
  nothing is on the other end. No error, no event.
- **Recorder** — the thing your reconcile loop calls. It needs a `*runtime.Scheme`
  to resolve the involved object's GroupVersionKind for `involvedObject`, and an
  `EventSource` naming the component, which is the `From` column in
  `kubectl get events`.

`recorder.Event(object, eventtype, reason, message)`: `eventtype` is
`corev1.EventTypeNormal` or `EventTypeWarning`; `reason` is a short CamelCase
machine-readable token (`Scheduled`, `FailedMount`) that people alert on, so keep
it stable; `message` is the human sentence. `Eventf` takes a format string.

**Key detail:** emission is **asynchronous**. `recorder.Event` returns
immediately and the broadcaster's goroutine does the POST, which is why the
exercise has to poll for the event to show up — and why `defer
broadcaster.Shutdown()` matters: exiting without it drops whatever is still in
flight.

The recorder also **aggregates**: identical events are deduplicated into a
single object with a rising `count` and an updated `lastTimestamp`, and it rate
limits per object. That's what stops a hot reconcile loop from writing millions
of events, and it means "I only see one event" is usually correct behaviour
rather than a bug.

Events are stored with a TTL (one hour by default) — they are a debugging aid,
not an audit log. Never build logic that depends on reading them back.

**References**

- `record` package: https://pkg.go.dev/k8s.io/client-go/tools/record
- Event API: https://pkg.go.dev/k8s.io/api/core/v1#Event
- Application introspection & debugging: https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/
