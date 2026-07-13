# setup

How every client-go program begins: locate the kubeconfig, build a
`*rest.Config`, wrap it in a clientset, and tune it for production.

## Resources

- [client-go on pkg.go.dev](https://pkg.go.dev/k8s.io/client-go)
- [clientcmd — kubeconfig loading](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd)
- [rest.Config reference](https://pkg.go.dev/k8s.io/client-go/rest#Config)
- [Accessing the API from a Pod vs outside](https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/)
