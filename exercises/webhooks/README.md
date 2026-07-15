# webhooks

Admission webhooks are just functions over an AdmissionReview. A validating
handler returns Allowed/Denied (echo the request UID, and remember Allowed
defaults to false). A mutating handler returns a JSON patch — and MUST declare
PatchType=JSONPatch, or the apiserver silently drops the patch.

These exercises test the handler logic directly — no cluster, no TLS wiring.

## Resources

- [Dynamic admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [admission/v1 API](https://pkg.go.dev/k8s.io/api/admission/v1)
- [AdmissionResponse](https://pkg.go.dev/k8s.io/api/admission/v1#AdmissionResponse)
- [JSON Patch (RFC 6902)](https://jsonpatch.com/)
