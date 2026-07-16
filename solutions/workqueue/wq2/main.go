// wq2
//
// Why do controllers back off exponentially? Because a resource that fails
// once often keeps failing, and hammering the API server helps no one. The
// exponential rate limiter answers When(key) with a delay that DOUBLES on
// every failure — base, 2·base, 4·base … capped at a max.
//
// Read the delays straight off the limiter and watch them grow.
package main

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	base := 5 * time.Millisecond
	max := time.Second

	rl := workqueue.NewTypedItemExponentialFailureRateLimiter[string](base, max)

	// Each When call records another failure for "x" and returns the delay the
	// queue would wait before redelivering it.
	d1 := rl.When("x")
	d2 := rl.When("x")
	d3 := rl.When("x")

	exkit.AssertEqual("1st failure waits the base delay", d1, base)
	exkit.AssertEqual("2nd failure doubles it", d2, 2*base)
	exkit.AssertEqual("3rd failure doubles again", d3, 4*base)

	exkit.Successf("exponential backoff: base·2^failures, capped at max — the queue's self-defense")
}
