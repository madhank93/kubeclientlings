# crds

Typed access to custom resources without generated code: converting between
unstructured maps and Go structs with runtime.DefaultUnstructuredConverter.
The converter maps by `json` tag in both directions, and the dynamic client
routes on the TypeMeta (apiVersion + kind) you set.

These exercises are pure conversion — no cluster required. (For serving a CRD
and CRUD through the dynamic client, see the `dynamic` topic.)

## Resources

- [runtime.UnstructuredConverter](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#UnstructuredConverter)
- [unstructured package](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured)
- [Custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [TypeMeta](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#TypeMeta)
