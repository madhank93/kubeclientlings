# dynamic

Life beyond typed clients: GroupVersionResource naming (resources are
lowercase plurals from the URL, Kinds live inside objects), reading
unstructured objects with the Nested* helpers, defining a CRD and talking
to it through the dynamic client, and asking the discovery API what the
server actually serves.

## Resources

- [dynamic package](https://pkg.go.dev/k8s.io/client-go/dynamic)
- [unstructured package](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured)
- [Custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [CRD versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [discovery package](https://pkg.go.dev/k8s.io/client-go/discovery)
