// wq1
//
// A rate-limiting queue tracks how many times each key has been requeued, so
// repeated failures back off. Two different calls "release" a key, and they
// are NOT the same:
//
//   Done   — "I stopped processing this item" (from Get/Done). Says nothing
//            about retries.
//   Forget — "clear this key's retry history" so its next failure starts from
//            the short base delay again, not a long one.
//
// A successful reconcile must reset the backoff so the next failure starts fresh.
//
// I AM NOT DONE
package main

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	rl := workqueue.NewTypedItemExponentialFailureRateLimiter[string](5*time.Millisecond, time.Second)
	q := workqueue.NewTypedRateLimitingQueue[string](rl)
	defer q.ShutDown()

	// Three failed reconciles of "a": each AddRateLimited bumps its retry count.
	q.AddRateLimited("a")
	q.AddRateLimited("a")
	q.AddRateLimited("a")
	exkit.AssertEqual("retries tracked after 3 rate-limited adds", q.NumRequeues("a"), 3)

	// The reconcile finally succeeds — but Done only releases the item from
	// processing; it does NOT touch the retry history. NumRequeues stays at 3,
	// so the next failure would start with a long backoff. Which call clears it?
	q.Done("a")
	exkit.AssertEqual("retries after Forget", q.NumRequeues("a"), 0)

	exkit.Successf("Forget resets backoff; Done just releases processing — reconcile success must Forget")
}
