# testing

Testing client-go code without a cluster: the fake clientset backed by an
in-memory tracker, reactors that inject API failures so you can exercise the
sad path, and Actions() inspection to assert that the right write actually
happened.

These exercises need no cluster — they run anywhere `go` does.

## Resources

- [fake clientset](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)
- [testing package (reactors, actions)](https://pkg.go.dev/k8s.io/client-go/testing)
- [ReactionFunc](https://pkg.go.dev/k8s.io/client-go/testing#ReactionFunc)
- [Unit testing controllers](https://book.kubebuilder.io/reference/envtest.html)
