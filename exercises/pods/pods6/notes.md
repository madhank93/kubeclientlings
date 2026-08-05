## pods6 — logs are a stream, and the container is not optional

```go
req := cs.CoreV1().Pods(ns).GetLogs("chatty", &corev1.PodLogOptions{
	Container: "writer", // required: this pod has more than one
})
stream, err := req.Stream(ctx)
defer stream.Close()
data, err := io.ReadAll(stream)
```

**Why it works**

- `GetLogs` does **no I/O**. It returns a `*rest.Request` describing
  `GET /api/v1/namespaces/<ns>/pods/chatty/log`. Nothing is sent until you call
  `Stream(ctx)`, which returns an `io.ReadCloser` over the open response body.
  That shape exists because a log is a pipe, not a value — with `Follow: true`
  the reader simply never reaches EOF.
- `defer stream.Close()` is mandatory. Every open log stream is a held HTTP
  connection on the apiserver *and* on the kubelet behind it; leaking them is a
  real way to exhaust both.
- `Container` defaults to the only container — but only when there **is** only
  one. With two, the apiserver refuses and tells you exactly what it wanted:
  `a container name must be specified for pod chatty, choose one of: [writer idle]`.
  This is a rare case of a loud, self-explaining Kubernetes error; most of this
  course is about the quiet ones.

**Key detail:** the option that matters most in an incident is **`Previous:
true`**. It reads the log of the *previous, dead* container instance, and it is
the only way to see why a `CrashLoopBackOff` pod died — the current instance's
log is empty or only shows the newest doomed attempt. `kubectl logs -p` is this
flag.

The rest of `PodLogOptions`, briefly:

- `Follow` — stream indefinitely. Pair it with a cancellable context, because
  nothing else will end the read.
- `TailLines` / `LimitBytes` — bound what you pull. A pod that has been up for
  a month can have gigabytes of log, and without a bound you will read all of
  it into memory.
- `SinceSeconds` / `SinceTime` — window the output.
- `Timestamps` — prefix each line with its RFC3339 time.

Note that logs are served from the **node**, proxied through the apiserver. If
the node is unreachable the request fails even though the pod object reads
perfectly — a distinction worth remembering when logs 500 but `kubectl get pod`
is fine.

**References**

- `PodLogOptions`: https://pkg.go.dev/k8s.io/api/core/v1#PodLogOptions
- `PodExpansion.GetLogs`: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#PodExpansion
- `rest.Request.Stream`: https://pkg.go.dev/k8s.io/client-go/rest#Request.Stream
- Debugging running pods: https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/
