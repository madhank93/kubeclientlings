## ctrl3 — leader election and its timing contract

```go
LeaseDuration:   15 * time.Second, // must be > RenewDeadline
RenewDeadline:   10 * time.Second, // must be > RetryPeriod * 1.2
RetryPeriod:     2 * time.Second,
ReleaseOnCancel: true,
```

`NewLeaderElector` enforces exactly two inequalities (plus "all three > 0"):

```go
if lec.LeaseDuration <= lec.RenewDeadline {
	return nil, fmt.Errorf("leaseDuration must be greater than renewDeadline")
}
if lec.RenewDeadline <= time.Duration(JitterFactor*float64(lec.RetryPeriod)) {
	return nil, fmt.Errorf("renewDeadline must be greater than retryPeriod*JitterFactor")
}
```

`JitterFactor` is `1.2`. So the second rule is not simply
`RenewDeadline > RetryPeriod` — retries are jittered up to 20% late, and the
deadline has to clear the *worst case* retry, not the nominal one. With 10s and
2s there is plenty of room; tighten the numbers and this is the check you trip.

**Why it works**

Run three replicas of a controller for availability and you must ensure only one
*acts*, or they fight over the same objects. Leader election solves it with a
single `coordination.k8s.io/v1` Lease object holding `holderIdentity` and
`renewTime` — whoever can write it is the leader.

The three durations describe one clock, seen from two sides:

- **LeaseDuration** — how long a lease stays valid after its last renewal. A
  *challenger* waits this long without seeing a renewal before declaring the
  seat vacant.
- **RenewDeadline** — how long the *incumbent* keeps trying to renew before
  giving up and calling `OnStoppedLeading`.
- **RetryPeriod** — how often either one retries its API call.

`LeaseDuration > RenewDeadline` is the safety property: the incumbent must stop
acting *before* anyone else is allowed to take over. Invert it — as the broken
version does with 5s and 10s — and there is a window where a healthy leader
still believes it leads while a challenger has already claimed the lease. Two
active controllers, which is exactly the outcome the whole mechanism exists to
prevent. That is the check `NewLeaderElector` refuses to build past.

**Under the hood**

- The elector loop is `wait.JitterUntil(tryAcquireOrRenew, RetryPeriod,
  JitterFactor, ...)`. A renew is a conditional `Update` of the Lease guarded by
  its `resourceVersion`, so two candidates writing at once means one gets a 409
  and loses — the lock is ordinary optimistic concurrency.
- A challenger compares `now` against `renewTime + LeaseDuration` using *its
  own* clock, which is why the durations must leave room for clock skew as well
  as for jitter.

**Common mistake**

- Writing an `OnStoppedLeading` that logs and returns. The process keeps
  reconciling while another replica already holds the lease, which is exactly the
  two-writer scenario leader election exists to prevent. Terminate.

**Key detail:** `OnStoppedLeading` must **terminate the process** (or at least
stop every reconcile loop), and it must do so fast. Losing the lease usually
means you were partitioned or stalled — another replica is already leading, and
anything you do from here on is a second writer. The conventional body is
`klog.Fatal("lost leadership")`; let the orchestrator restart you as a
candidate.

`ReleaseOnCancel: true` makes a clean shutdown *clear* the lease instead of
letting it expire, so failover on a rolling update takes milliseconds rather
than a full `LeaseDuration`.

Defaults worth knowing: kube-controller-manager ships 15s / 10s / 2s — the same
numbers used here. Shorter values fail over faster but tolerate less latency;
too short and a GC pause or a slow API server costs you the leadership you
already held.

Also note this is a **general** distributed lock, not a controller-only tool —
anything needing "exactly one instance active" can use the same Lease.
`resourcelock.LeaseLock` is the modern lock type; the older ConfigMap and
Endpoints locks are deprecated.

**See also:** ctrl2 (the controller that should only run when leading) · pods3 (the same
optimistic concurrency underneath the Lease) · the
[controllers chapter](../README.md)

**References**

- `leaderelection` package: https://pkg.go.dev/k8s.io/client-go/tools/leaderelection
- Lease API: https://kubernetes.io/docs/concepts/architecture/leases/
- `resourcelock`: https://pkg.go.dev/k8s.io/client-go/tools/leaderelection/resourcelock
