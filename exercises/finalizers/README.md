# finalizers

The operator's cleanup hook. While a finalizer is present, Delete does not
remove the object — the apiserver stamps it with a DeletionTimestamp and
leaves it terminating until the finalizer is cleared. Adding the finalizer
protects the object; removing it (after cleanup) is what finally lets it go.

## Resources

- [Finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
- [Using finalizers to control deletion](https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/)
- [ObjectMeta.Finalizers](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ObjectMeta)
- [Foreground vs background deletion](https://kubernetes.io/docs/concepts/architecture/garbage-collection/)
