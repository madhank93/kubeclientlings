# Capstone projects design

Date: 2026-08-06
Status: approved for planning

## Problem

The 56 exercises teach client-go primitives one at a time: a Get here, a
workqueue there, a finalizer somewhere else. Nothing assembles them. A learner
finishes the set able to explain `AddAfter` and still unable to write the
controller their platform team actually runs.

Capstones close that gap. Each one is a job a platform engineer is really
asked for, built in stages, ending in a controller the learner wrote every
line of.

## Scope

Three capstones, 16 exercises, in a new `exercises/capstones/` tree:

| Capstone | Directory | Stages | The job |
|---|---|---|---|
| Hibernator | `capstones/hibernate` | `hib1`–`hib6` | Scale non-prod workloads to zero outside business hours; restore them at the start of the next window |
| Janitor | `capstones/janitor` | `jan1`–`jan5` | Reap ephemeral preview namespaces once they pass their expiry |
| Propagator | `capstones/propagate` | `pro1`–`pro5` | Keep a golden ConfigMap present and correct in every opted-in namespace |

Deferred to a later pass, once these land: CRD-driven policy objects, status
conditions, a metrics endpoint, and admission-webhook validation of schedules.

## Decisions

**Staged chains in the existing format.** Every stage is one self-verifying
`main.go` with one planted bug plus a `notes.md`, exactly like the 56
exercises before it. No new runner, no new file layout, no changes to `reset`,
`hint`, the TUI or the web catalog.

**Cumulative skeletons.** Stage N's solution, plus a planted bug and the
`I AM NOT DONE` marker, is stage N+1's exercise file. `hib1` is ~80 lines of
schedule math; `hib6` is a ~300-line controller. The chain has a build-order
dependency, which is an authoring concern, not a learner-facing one.

**Injected clocks, not wall time.** Exercises take a `clock.Clock` rather than
calling `time.Now()`. This mirrors real controllers — kubelet, the
kube-controller-manager and client-go's own rate limiters all take a clock —
and it makes the lesson enforceable: code that calls `time.Now()` directly
fails against the harness's fake clock.

`workqueue.TypedDelayingQueueConfig` accepts a `Clock clock.WithTicker`, and
`*clocktesting.FakeClock` satisfies that interface. One fake clock therefore
drives both the policy decision and the queue's `AddAfter` delays: stepping
the clock fires the requeue. `FakeClock.HasWaiters()` reports when the loop
has parked on its timer, so the harness steps only after the loop is waiting
and the sequence is deterministic.

**Raw client-go, with one controller-runtime finale.** Informers, workqueues
and typed clients throughout — the capstone is where the earlier domains
combine. `hib6` re-implements the finished hibernator on controller-runtime so
the learner sees precisely which parts Manager and Builder hide.

**One-shot programs.** An exercise starts its loop, advances the fake clock,
waits for the cluster to settle, asserts, and exits. No stage blocks forever.

## Stages

### Hibernator

| Stage | Builds | Planted bug | Teaches |
|---|---|---|---|
| `hib1` | Parse `hibernate.kubeclientlings.dev/schedule: "Mon-Fri 09:00-18:00 Asia/Kolkata"`; decide awake or asleep | Calls `time.Now()` in the machine's local zone instead of the injected clock and the policy's `time.Location` | Why controllers take a clock; untestable time |
| `hib2` | Scale to zero through the scale subresource; record the original replica count in an annotation | Scales first and records second, so a crash between the two loses the count forever | Write ordering as durability; `UpdateScale` |
| `hib3` | Wake: restore from the annotation, then clear it | Treats a missing annotation as zero, and overwrites a human who scaled up during the window | Level-triggered idempotency; absent is not zero |
| `hib4` | Drive the loop from an informer and a workqueue | Reacts only to watch events, so nothing happens at 18:00 — no event fires when a boundary passes | `AddAfter` to the next boundary; resync is not a timer |
| `hib5` | Dry run, protected-namespace skip, Events, status | `Update` on a stale object clobbers a concurrent writer | Conflict handling; rolling out a destructive action safely |
| `hib6` | The same loop on controller-runtime | Returns `ctrl.Result{}` with no `RequeueAfter` | What Manager and Builder hide |

`hib4` carries the chain. A scheduled controller that is purely event-driven
passes every test and silently never fires in production.

### Janitor

| Stage | Planted bug | Teaches |
|---|---|---|
| `jan1` | Lists every namespace and filters client-side; parses the `expiresAt` annotation with the wrong RFC3339 layout | Label selectors are server-side work; `time.Parse` layouts |
| `jan2` | Runs the protection check after the delete call, and matches by name prefix so `kube-system-test` slips through | Ordering of safety checks; deny-lists are traps |
| `jan3` | Implements "dry run" by skipping the call client-side, so admission never sees it | `DryRun: []string{metav1.DryRunAll}`; `PropagationPolicy` |
| `jan4` | Rewrites the grace-start stamp every pass, so the grace period never expires | Drift of "now"; Events as the audit trail |
| `jan5` | Deletes with unbounded parallelism and ignores 429 | Pagination; workqueue rate limiting; backoff |

### Propagator

| Stage | Planted bug | Teaches |
|---|---|---|
| `pro1` | `Create` only, so the second pass fails with `AlreadyExists` | Apply as the fan-out primitive |
| `pro2` | Field-manager name varies per run, piling up managed fields; `Update` strips other managers' fields | Server-Side Apply field ownership |
| `pro3` | Watches ConfigMaps only, so a new namespace never receives a copy | Watch what creates work, not only what you write |
| `pro4` | Prunes copies via a cross-namespace `ownerReference` — accepted by the API, never acted on by GC | ownerRefs are namespace-local; label-based tracking |
| `pro5` | Reconcile wedges on a conflict when someone hand-edits a copy | `Force` on Apply; drift repair |

`pro4` ships to production regularly: the API accepts the ownerRef and nothing
ever reports that it is inert.

## Safety

`jan2` plants a broken protection check and the propagator writes across
namespaces — on the learner's own cluster. Two guardrails, both in `exkit`,
which is always correct:

1. Exercises select only namespaces labeled with the existing
   `exkit.ExerciseLabel` (`kubeclientlings.dev/exercise=<name>`), applied by
   the harness when it creates them — the same label `mise run nuke` already
   uses to find exercise namespaces.
2. `exkit.GuardedDelete` refuses any namespace without the `clx-` prefix and
   fails the exercise loudly.

A learner's buggy attempt at `jan2` therefore produces a clear failure instead
of deleting something real.

## exkit additions

```go
func FakeClockAt(rfc3339 string) *clocktesting.FakeClock
func WaitForWaiters(clk *clocktesting.FakeClock, n int)
func PreviewNamespace(ctx context.Context, cs kubernetes.Interface, exercise, suffix string, ann map[string]string) string
func GuardedDelete(ctx context.Context, cs kubernetes.Interface, ns string) error
func WaitForReplicas(ctx context.Context, cs kubernetes.Interface, ns, name string, want int32)
func WaitForEvent(ctx context.Context, cs kubernetes.Interface, ns, reason string)
```

`k8s.io/utils` moves from an indirect to a direct dependency for
`clock` and `clock/testing`. No new module.

`internal/exkit` has no tests today. The guard and the clock helper get unit
tests, because a broken guard is a cluster-safety bug rather than a teaching
bug.

## Verification

Each stage self-verifies against the local kind cluster by exit code, as every
other exercise does. `hib1` needs no cluster at all and runs in under a
second. Stages carry `timeout = 240`, matching the heaviest existing
exercises.

Namespace deletion in kind is slow, so janitor stages run about 45 seconds
each. Sixteen new exercises take `mise run e2e` from roughly 9 minutes toward
16. `kubeclientlings verify --domain capstones` is added so authoring does not
require a full-set run.

## Authoring order

Solutions are written first, in stage order, because each one is the next
stage's starting point. `hack/capstone-diff.sh <capstone>` prints stage N's
solution against stage N+1's exercise, so the chain cannot drift silently as
stages are edited.

Each capstone's first stage opens with a `## The job` comment block describing
the real-world task in a paragraph, so the learner knows what they are
building before line one.

## Implementation sequencing

Sixteen exercises is more than one plan should carry, and the three capstones
are independent once the foundation exists. Implementation splits into four
plans, in order:

1. **Foundation** — the `exkit` additions, their unit tests, the
   `k8s.io/utils` promotion, `verify --domain`, and `hack/capstone-diff.sh`.
2. **Hibernator** — `hib1`–`hib6`, the only chain that exercises the fake
   clock end to end, so it validates the foundation.
3. **Janitor** — `jan1`–`jan5`.
4. **Propagator** — `pro1`–`pro5`.

Each plan ends green: lint clean, unit tests passing, and every exercise in
that plan verified against the kind cluster by exit code.

## Out of scope

- CRD-driven policy objects, status conditions, metrics endpoints and
  schedule-validating webhooks — a later pass, once these three land.
- The remaining five candidate capstones considered and deferred: secret
  rotation with rolling restart, canary rollout with auto-rollback, a
  rightsizer over the metrics API, CR backup and restore, and a PDB-aware
  drain orchestrator.
