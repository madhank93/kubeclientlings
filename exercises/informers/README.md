# informers

The caching layer every controller is built on: SharedInformerFactory
lifecycle (Start after materializing informers, then WaitForCacheSync),
event handlers that push instead of poll, cache keys as namespace/name via
MetaNamespaceKeyFunc, and scoping the List+Watch with factory options.

## Resources

- [informers package](https://pkg.go.dev/k8s.io/client-go/informers)
- [tools/cache package](https://pkg.go.dev/k8s.io/client-go/tools/cache)
- [Controller pattern](https://kubernetes.io/docs/concepts/architecture/controller/)
- [sample-controller walkthrough](https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md)
