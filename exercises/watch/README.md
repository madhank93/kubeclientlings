# watch

Watching the cluster change: event types on the watch channel (Added /
Modified / Deleted — check the constants, not hand-written strings), what
resourceVersion a watch starts from and why list-then-watch is the correct
pattern, RetryWatcher for streams that survive dropped connections, and the
EventBroadcaster/EventRecorder machinery behind `kubectl describe` events.

## Resources

- [Efficient detection of changes](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
- [watch package](https://pkg.go.dev/k8s.io/apimachinery/pkg/watch)
- [RetryWatcher](https://pkg.go.dev/k8s.io/client-go/tools/watch#RetryWatcher)
- [record package (EventBroadcaster)](https://pkg.go.dev/k8s.io/client-go/tools/record)
