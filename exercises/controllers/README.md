# controllers

The controller pattern assembled from its parts: the workqueue Get/Done
contract (and per-key dedup), the informer → key → queue → worker pipeline
that is every controller's skeleton, and leader election with Lease locks
so only one replica reconciles at a time.

## Resources

- [workqueue package](https://pkg.go.dev/k8s.io/client-go/util/workqueue)
- [Controller pattern](https://kubernetes.io/docs/concepts/architecture/controller/)
- [sample-controller](https://github.com/kubernetes/sample-controller)
- [leaderelection package](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection)
- [Coordinated leader election](https://kubernetes.io/docs/concepts/cluster-administration/coordinated-leader-election/)
