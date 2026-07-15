# ssa

Server-Side Apply: declare the state you want and let the apiserver merge it,
while it tracks which field manager owns each field. Every apply needs a
FieldManager, and a second manager touching an owned field conflicts — Force
is how you knowingly take ownership.

## Resources

- [Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
- [applyconfigurations](https://pkg.go.dev/k8s.io/client-go/applyconfigurations)
- [ApplyOptions](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ApplyOptions)
- [Field management & conflicts](https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts)
