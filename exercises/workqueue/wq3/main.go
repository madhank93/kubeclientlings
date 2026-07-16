// wq3
//
// A poison item — one that fails no matter what — must not requeue forever.
// Every real controller has a handleErr that caps retries: requeue with
// backoff while NumRequeues is under a limit, and once it hits the limit,
// Forget the key and drop it so it stops consuming the queue.
//
// Implement the retry cap so a permanently failing key is eventually dropped.
//
// I AM NOT DONE
package main

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

const maxRetries = 3

func main() {
	rl := workqueue.NewTypedItemExponentialFailureRateLimiter[string](time.Millisecond, 10*time.Millisecond)
	q := workqueue.NewTypedRateLimitingQueue[string](rl)
	defer q.ShutDown()

	// The controller error handler — but this one ALWAYS requeues. With no
	// cap and no Forget, a key that fails forever is requeued forever. Gate
	// the requeue on NumRequeues(key) < maxRetries, and Forget past the cap.
	handleErr := func(key string) {
		q.AddRateLimited(key)
	}

	// "bad" fails every time. Drive handleErr up to the cap, then once more.
	for q.NumRequeues("bad") < maxRetries {
		handleErr("bad")
	}
	handleErr("bad")

	exkit.AssertEqual("a permanently failing key is dropped after maxRetries", q.NumRequeues("bad"), 0)

	exkit.Successf("requeue under the cap, Forget at the cap — that is how a controller survives poison items")
}
