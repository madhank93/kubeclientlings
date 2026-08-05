## deploy4 — what "the rollout is done" actually means

```go
if d.Status.ObservedGeneration < d.Generation {
	return false, nil // status is stale — describes an older spec
}
return d.Status.UpdatedReplicas == replicas && d.Status.ReadyReplicas == replicas, nil
```

**Why it works**

Three different counters live in `DeploymentStatus`, and they answer three
different questions:

- `Status.Replicas` — pods that **exist** under this Deployment. Includes old
  pods being drained and new pods still starting. Almost never what you want.
- `Status.UpdatedReplicas` — pods created from the **current** pod template.
  Answers "has the new version been rolled out to everything?"
- `Status.ReadyReplicas` — pods passing their readiness probe. Answers "can
  they serve traffic?"
- `Status.AvailableReplicas` — ready **and** stable for
  `minReadySeconds`. Stricter still.

The generation check comes first and is the one people forget.
`metadata.generation` increments on every spec change; the controller copies it
into `status.observedGeneration` once it has acted. Reading status before that
happens gives you the *previous* rollout's numbers, which very often look
complete — so the wait returns instantly and you assert against a version that
was never deployed.

**Key detail:** this is exactly the logic `kubectl rollout status` implements,
and it generalises to every controller that has a status. Whenever you poll a
`status` block, check `observedGeneration` against `generation` first; only then
trust the rest.

For failure detection rather than success, read
`Status.Conditions`: a `Progressing` condition with
`reason: ProgressDeadlineExceeded` is how a wedged rollout announces itself, and
without it your wait just runs to timeout with no explanation.

**References**

- `DeploymentStatus`: https://pkg.go.dev/k8s.io/api/apps/v1#DeploymentStatus
- Deployment status & conditions: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status
- The `generation` / `observedGeneration` pattern: https://kubernetes.io/docs/reference/using-api/api-concepts/
