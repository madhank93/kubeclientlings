## wh1 — a validating webhook is just a function

```go
resp := &admissionv1.AdmissionResponse{UID: review.Request.UID} // echo the UID

if _, ok := pod.Labels["team"]; !ok { // deny when the label is ABSENT
	resp.Allowed = false
	resp.Result = &metav1.Status{Message: "pods must carry a team label"}
	return resp
}

resp.Allowed = true // must be explicit — the zero value is false
return resp
```

**Why it works**

- Admission control looks intimidating (TLS certs, CA bundles,
  `ValidatingWebhookConfiguration`) but the part you write is one pure function:
  `AdmissionReview` in, `AdmissionResponse` out. All of it is unit-testable with
  no cluster, which is exactly what this exercise does.
- `review.Request.Object.Raw` is the object's raw JSON — the API server does not
  send you a typed struct, because it cannot know your Go types. Decode it
  yourself with `json.Unmarshal`.
- **Echo the UID.** The API server matches responses to requests by
  `response.uid == request.uid`. Get it wrong (or leave it empty) and the
  request fails with "expected response.uid to match" — regardless of what you
  decided.
- **`Allowed` is a bool, so it defaults to `false`.** Every allow path must set
  it explicitly. Forgetting is fail-closed rather than fail-open, which is the
  right way round for a security control, but it does mean a webhook that
  returns early on some branch will block everything that reaches it.
- `Result.Message` is what the user sees when their `kubectl apply` is
  rejected — write it for them, not for your logs. `Result.Code` sets the HTTP
  status.

**Key detail:** the parts *around* the function are where webhooks actually
bite in production.

- **`failurePolicy`** on the webhook configuration decides what happens when
  your endpoint is down. `Fail` (the default) means every matching API request
  is rejected — a crashed webhook that matches `pods` can stop the entire
  cluster from scheduling. Scope `rules` and `namespaceSelector` tightly, and
  always exclude `kube-system`.
- **`timeoutSeconds`** defaults to 10 and counts against every matching request.
  Never call out to another service from inside a webhook.
- Webhooks must be **idempotent and side-effect free**: the API server may call
  them more than once for one request (notably during a dry-run).

Since Kubernetes 1.30, simple policies like this one are often better expressed
as a **ValidatingAdmissionPolicy** with a CEL expression — no server, no certs,
no availability risk. Reach for a webhook when you need real code.

**References**

- Dynamic admission control: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
- `admissionv1` types: https://pkg.go.dev/k8s.io/api/admission/v1
- Validating admission policy: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
