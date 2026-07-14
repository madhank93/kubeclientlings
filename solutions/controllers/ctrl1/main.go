// ctrl1
//
// The workqueue is the beating heart of every controller. It has a
// contract: Get hands you an item and marks it PROCESSING; you MUST call
// Done when finished. If the same key is Added while you're processing it,
// the queue holds it back — and re-delivers only after your Done. Skip
// Done and that key is stuck forever: Get never returns it again.
//
// Honor the Get/Done contract so the requeued item comes back.
package main

import (
	"strings"
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	_, cancel, _, _ := exkit.Begin("ctrl1")
	defer cancel()

	q := workqueue.NewTyped[string]()
	defer q.ShutDown()

	// "a" added twice while pending — the queue dedupes it.
	q.Add("a")
	q.Add("b")
	q.Add("a")
	exkit.AssertEqual("queue length after Add(a), Add(b), Add(a)", q.Len(), 2)

	processed := make(chan []string, 1)
	go func() {
		var got []string
		for i := range 3 {
			item, shutdown := q.Get()
			if shutdown {
				return
			}
			got = append(got, item)
			if i == 0 {
				// Something re-queues the key WHILE it is being processed —
				// exactly what happens when an object changes mid-reconcile.
				q.Add(item)
			}
			// The other half of the contract: processing is finished, and
			// the held-back copy (if any) may now be delivered.
			q.Done(item)
		}
		processed <- got
	}()

	select {
	case got := <-processed:
		exkit.AssertEqual("processing order", strings.Join(got, ","), "a,b,a")
	case <-time.After(10 * time.Second):
		exkit.Failf("the third Get never returned — a key re-added during processing only comes back after Done")
	}

	exkit.Successf("Get marks processing, Done releases — that contract is what makes reconcile loops safe")
}
