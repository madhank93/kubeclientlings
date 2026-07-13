# services

Services and their backends: the selector → pod contract (silently broken
selectors are legal!), and reading EndpointSlices from the discovery.k8s.io/v1
API via the well-known service-name label.

## Resources

- [Service concepts](https://kubernetes.io/docs/concepts/services-networking/service/)
- [EndpointSlices](https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/)
- [discovery/v1 API](https://pkg.go.dev/k8s.io/api/discovery/v1)
