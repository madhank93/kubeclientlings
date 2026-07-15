# subresources

Some fields live behind their own endpoint. Replicas are set through /scale
(Scale.Spec.Replicas is the desired count; Status.Replicas is read-only), and
when the status subresource is enabled, status becomes a separate write path —
the main endpoint ignores status, and only UpdateStatus can move it.

## Resources

- [Scale subresource](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/deployment-v1/#DeploymentSpec)
- [CRD status subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)
- [UpdateStatus vs Update](https://pkg.go.dev/k8s.io/client-go/dynamic#ResourceInterface)
- [autoscaling Scale type](https://pkg.go.dev/k8s.io/api/autoscaling/v1#Scale)
