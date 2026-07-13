# deployments

The apps/v1 client: creating Deployments (selector ↔ template contract),
scaling via `*int32`, patching container images with merge keys, and knowing
when a rollout is actually done (spec vs status, generations, readiness).

## Resources

- [AppsV1 DeploymentInterface](https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface)
- [Deployment status fields](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status)
- [Strategic merge patch and merge keys](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/)
