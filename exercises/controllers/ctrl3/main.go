// ctrl3
//
// Run two replicas of a controller and they fight — unless exactly one
// leads. Leader election settles it with a Lease object. The timings form
// a contract the library ENFORCES: LeaseDuration (how long a lost leader's
// lease lingers) must be longer than RenewDeadline (how long the leader
// keeps trying to renew), which must be longer than RetryPeriod.
//
// Fix the timing contract so the candidate can run for office.
//
// I AM NOT DONE
package main

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("ctrl3")
	defer cancel()

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: "kubeclientlings-leader", Namespace: ns},
		Client:     cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: "candidate-1"},
	}

	becameLeader := make(chan struct{})
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: lock,
		// A lease that expires BEFORE the leader even stops trying to
		// renew it is nonsense — a healthy leader would keep losing its
		// own seat. The library refuses this config outright.
		LeaseDuration:   5 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) { close(becameLeader) },
			OnStoppedLeading: func() {},
		},
	})
	if err != nil {
		exkit.Failf("building the leader elector: %v", err)
	}

	elCtx, elCancel := context.WithCancel(ctx)
	defer elCancel()
	go elector.Run(elCtx)

	select {
	case <-becameLeader:
	case <-time.After(30 * time.Second):
		exkit.Failf("never became leader")
	}

	lease, err := cs.CoordinationV1().Leases(ns).Get(ctx, "kubeclientlings-leader", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("reading the lease: %v", err)
	}
	if lease.Spec.HolderIdentity == nil {
		exkit.Failf("lease has no holder")
	}

	exkit.AssertEqual("the lease holder", *lease.Spec.HolderIdentity, "candidate-1")
	exkit.Successf("elected. The Lease object is the whole ballot box — go look: kubectl -n %s get lease", ns)
}
