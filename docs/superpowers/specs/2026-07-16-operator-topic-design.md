# Design: `operator` topic — controller-runtime exercises on kind

Date: 2026-07-16
Status: approved

## Context

The v2-spec harvest deliberately skipped controller-runtime, judging it a poor
fit for the single-file `go run` + exkit model. Revisited: a real
`Manager`/`For`/`Owns` controller fits a single `main.go` fine — start the
manager in a goroutine, drive objects with the exkit clientset, assert the
reconciler's side effects with `exkit.WaitFor`. All exercises run against the
local kind cluster, matching the existing `controllers` (ctrl1-3) topic.

The other deferred harvest item — a justfile over the existing CLI — is
**dropped**: mise.toml is already the task runner and a second runner would
only drift.

## Scope

- New topic dir `operator/` under both `exercises/` and `solutions/`.
- Three exercises: `op1`, `op2`, `op3`. Single-file `main.go` each, broken
  version carries the `I AM NOT DONE` marker + inline hint comment; solution
  is the fixed twin.
- New dependency: `sigs.k8s.io/controller-runtime`, version compatible with
  the k8s.io/* versions already in go.mod.
- Registered in `info.toml` with `mode = "compile"` and hints, after the
  `controllers` topic. Splash exercise count is computed live — no manual
  update.

## Non-goals

- No CRDs, no envtest, no multi-file operator scaffold, no kubebuilder.
- No exkit changes: exkit stays pure client-go. Manager construction
  (metrics disabled, leader election off, cache scoped to the exercise
  namespace) lives visibly in each exercise — the wiring *is* the lesson.

## Exercises

### op1 — the Reconcile contract

Manager with `For(&corev1.ConfigMap{})`, cache scoped to the exercise
namespace. Reconciler fetches the object via `mgr.GetClient()` and patches
label `reconciled=true`.

- **Bug (broken version):** reconciler returns the raw error when the object
  is gone — `Get` NotFound is treated as failure, so deletes put the
  controller in an error-requeue loop, and the fetch uses a hand-built key
  instead of `req.NamespacedName`.
- **Teaches:** request in → `ctrl.Result` out; NotFound means "object gone,
  return nil"; the manager client is cache-backed.
- **Assert:** create ConfigMap, `WaitFor` label. The reconciler records
  unexpected errors in an atomic counter; after deleting the ConfigMap and
  letting the delete-event reconcile fire, the exercise asserts the counter
  is zero. Broken version: wrong key means the label never appears
  (`WaitFor` timeout) and NotFound-as-error increments the counter.

### op2 — Owns and ownership

Reconciler of a parent ConfigMap ensures a child ConfigMap exists, with
`Owns(&corev1.ConfigMap{})` on the builder so child events re-trigger the
parent's reconcile.

- **Bug:** missing `controllerutil.SetControllerReference` — the child has no
  ownerRef, so deleting it never maps back to the parent and it is never
  recreated.
- **Teaches:** ownerRefs are how `Owns` maps child events to parent requests;
  GC and self-healing both hang off them.
- **Assert:** child appears with controller ownerRef → exkit deletes child →
  `WaitFor` recreation.

### op3 — CreateOrUpdate idempotency

Reconciler ensures a child object using `controllerutil.CreateOrUpdate` with
a mutate callback.

- **Bug:** desired state is set on the object *before* the call instead of
  inside the mutate fn (blind Create), so the second reconcile hits
  AlreadyExists / stomps concurrent changes and errors.
- **Teaches:** mutate-fn pattern — read-modify-write that is correct on both
  create and update paths.
- **Assert:** object converges; exkit mutates the child, `WaitFor` reconciler
  restoring desired state without error churn.

## Verification

- `mise run build`, `mise run test`, `mise run lint` green.
- Targeted verify of op1-3 against kind.
- Full `mise run e2e` (solutions overlaid onto exercises) green, modulo the
  known pre-existing ctrl3 teardown flake.

## Follow-through

- Update `v2-spec-harvest` memory: both deferred infra items resolved
  (operator built, justfile dropped).
