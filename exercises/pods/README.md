# pods

Typed CRUD against the core/v1 API: Create, Get, List (with server-side
filtering), Update (optimistic concurrency), Patch (merge semantics), and
Delete (async, confirmed via NotFound).

## Resources

- [CoreV1 PodInterface](https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface)
- [Label selector syntax](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)
- [Update vs Patch — API conventions](https://kubernetes.io/docs/reference/using-api/api-concepts/#patch-and-apply)
- [apierrors helpers](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/errors)
