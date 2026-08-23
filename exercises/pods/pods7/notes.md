## pods7 — exec is a protocol upgrade, and stdout is requested twice

```go
req := cs.CoreV1().RESTClient().Post().
	Resource("pods").Name("shell").Namespace(ns).
	SubResource("exec").
	VersionedParams(&corev1.PodExecOptions{
		Container: "app",
		Command:   []string{"sh", "-c", "echo " + marker},
		Stdout:    true, // ask the SERVER to send it
		Stderr:    true,
	}, scheme.ParameterCodec)

executor, err := remotecommand.NewSPDYExecutor(exkit.MustRESTConfig(), "POST", req.URL())

var stdout, stderr bytes.Buffer
err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
	Stdout: &stdout, // say where to PUT it locally
	Stderr: &stderr,
})
```

**Why it works**

- There is no `Exec()` on the typed clientset, because exec is not a
  request/response — it is an HTTP connection **upgraded** to SPDY and then
  multiplexed into separate stdin/stdout/stderr channels. The typed clients only
  generate ordinary REST verbs, so you drop to `RESTClient()` and build the
  request by hand.
- `VersionedParams(..., scheme.ParameterCodec)` encodes `PodExecOptions` into
  the **query string** (`?container=app&command=sh&stdout=true&…`), not a body.
  That is what a `ParameterCodec` is for, and why the options type is registered
  in the scheme.
- `NewSPDYExecutor` takes the `*rest.Config`, not the clientset: it builds its
  own connection and needs the raw TLS and auth material to do it.

**Under the hood**

- `NewSPDYExecutor` performs an HTTP GET/POST with `Connection: Upgrade` and
  `Upgrade: SPDY/3.1`. The apiserver proxies the upgraded connection to the
  kubelet, which multiplexes it into numbered streams — one per channel that was
  requested, plus an error stream carrying the exit status.
- Only the channels named in `PodExecOptions` are opened, which is why the local
  `StreamOptions` writers are a separate decision: they attach to streams that
  may or may not exist.

**Common mistake**

- Attaching a local `StreamOptions.Stdout` without setting
  `PodExecOptions.Stdout`. The executor waits on a stream the server never
  negotiated and the session dies with a bare `Timeout occurred` that names
  neither. Set both, or neither.

**Key detail:** stdout appears in both structs and they are **not** redundant.

- `PodExecOptions.Stdout` goes on the wire and tells the apiserver *whether to
  open that channel at all*.
- `StreamOptions.Stdout` is purely local — where the bytes land once they
  arrive.

Set neither (the broken version) and the command runs fine, the session closes
cleanly, and your buffer is simply empty — a silent no-output.

Set exactly **one** of them and it is worse. Attaching a local
`StreamOptions.Stdout` writer without asking for `PodExecOptions.Stdout` makes
the executor wait on a channel the apiserver never negotiated, and the session
hangs until it dies with a bare:

```
streaming the exec session: Timeout occurred
```

— an error that names neither stdout nor the option you missed. If an exec
mysteriously times out while the same command works under `kubectl exec`, this
mismatch is the first thing to check. Set both, or neither.

Two more details worth carrying:

- Use **`StreamWithContext`**, not `Stream`. `Stream` is explicitly deprecated
  because it cannot be cancelled, so a wedged session leaks the connection and
  its goroutines forever.
- `Command` is an **argv array executed directly** — there is no shell. So
  `[]string{"echo", "$HOME"}` prints the literal `$HOME`. Wrap it yourself with
  `{"sh", "-c", "..."}` when you need globbing, pipes or variables.

With `TTY: true` stdout and stderr are merged into one stream (a real terminal
has only one), so a `Stderr` writer is then invalid — that asymmetry catches
people writing interactive shells.

Newer clusters also support a WebSocket transport
(`NewWebSocketExecutor`); `NewFallbackExecutor` tries it and falls back to SPDY,
which is what current `kubectl` does.

**See also:** pods6 (the simpler streaming subresource) · setup1 (the `*rest.Config` the
executor needs) · the [pods chapter](../README.md)

**References**

- `remotecommand`: https://pkg.go.dev/k8s.io/client-go/tools/remotecommand
- `PodExecOptions`: https://pkg.go.dev/k8s.io/api/core/v1#PodExecOptions
- `rest.Request.VersionedParams`: https://pkg.go.dev/k8s.io/client-go/rest#Request.VersionedParams
- Get a shell to a running container: https://kubernetes.io/docs/tasks/debug/debug-application/get-shell-running-container/
