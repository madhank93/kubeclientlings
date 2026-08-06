## setup3 — tuning a rest.Config for production

```go
config.QPS = 50
config.Burst = 100
config.Timeout = 10 * time.Second
config.UserAgent = "kubeclientlings/setup3"
```

**Why it works**

- `QPS` / `Burst` drive a **client-side** token-bucket rate limiter that sits in
  front of the transport. The defaults are 5 QPS / 10 burst, which is fine for
  a CLI making a handful of calls and disastrous for a controller: once the
  bucket is empty client-go *silently blocks* before the request is even sent.
  Symptom: multi-second latency with a completely idle API server. Raise both,
  and keep `Burst >= QPS` or a burst is throttled the instant it starts.
- `Timeout` is applied to the underlying `http.Client`, so it bounds the whole
  request. Without it, a wedged API server or a black-holed connection hangs
  the calling goroutine forever.
- `UserAgent` is sent on every request and lands in the API server's audit log
  and metrics. When a cluster is being hammered, this string is how an operator
  identifies *which* client is responsible. Set it to something that names your
  binary and version.

**Key detail:** `Timeout` is a blunt whole-request deadline and it does **not**
belong on a watch or on any long-poll client — it will tear the stream down on
schedule. Give informer/watch clients their own untimed config (or a much
larger one) and rely on `context` deadlines for individual calls. Likewise, all
of these fields must be set *before* `kubernetes.NewForConfig`: the constructor
copies the config, so mutating it afterwards changes nothing.

Newer client-go also honours server-side **API Priority and Fairness**, which
is why upstream now often recommends raising client QPS well above the old
defaults and letting the server do the fair queueing.

**References**

- `rest.Config` fields: https://pkg.go.dev/k8s.io/client-go/rest#Config
- Rate limiting in client-go: https://pkg.go.dev/k8s.io/client-go/util/flowcontrol
- API Priority and Fairness: https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
