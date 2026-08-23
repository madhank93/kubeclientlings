## wq2 — exponential backoff, and the limiters that aren't

```go
rl := workqueue.NewTypedItemExponentialFailureRateLimiter[string](base, max)

d1 := rl.When("x") // base
d2 := rl.When("x") // 2·base
d3 := rl.When("x") // 4·base
```

**Why it works**

- `When(item)` is the rate limiter's whole interface: it records one more
  failure for that key and returns how long the queue should wait before
  redelivering it. `AddRateLimited` is just `AddAfter(item, rl.When(item))`.
- The exponential limiter returns `base · 2^failures`, clamped at `max`. With a
  5ms base that's 5ms, 10ms, 20ms, 40ms… A resource that fails once usually
  keeps failing, and doubling means a broken object costs a handful of attempts
  per minute instead of thousands.
- Counters are **per key**. One wedged object backing off does not slow down
  reconciles of anything else — which is exactly what you want and is easy to
  lose if you implement backoff yourself with a single global timer.
- `Forget(item)` (wq1) resets that key's counter, so the exponent only grows
  while the key is actually failing.

**Under the hood**

- `ItemExponentialFailureRateLimiter` keeps `map[item]int` of failures behind a
  mutex. `When` reads the count, computes `baseDelay * 2^n` with an overflow
  guard, clamps to `maxDelay`, then increments. `Forget` deletes the entry.
- `AddRateLimited` is literally `AddAfter(item, rl.When(item))`, and `AddAfter`
  parks the item on a heap-ordered waiting loop until its time arrives.

**Common mistake**

- Reaching for `FastSlow` because the name sounds like backoff. It returns a
  flat delay for the first N attempts and a different flat delay afterwards, so
  a permanently broken object retries at a constant rate forever instead of
  backing away.

**Key detail:** the limiters look interchangeable and are not.

- `NewTypedItemExponentialFailureRateLimiter(base, max)` — doubles per failure.
  The one you want for reconcile errors.
- `NewTypedItemFastSlowRateLimiter(fast, slow, maxFastAttempts)` — returns a
  **flat** `fast` delay for the first N attempts, then a flat `slow` delay. No
  doubling at all, which is why the "doubles" assertions fail against it. It
  suits "poll quickly for a bit, then settle down", not error backoff.
- `NewTypedMaxOfRateLimiter(...)` composes several and takes the largest.
- `NewTypedBucketRateLimiter(rate.NewLimiter(...))` is a *global* token bucket,
  not per key — it caps total throughput.

The default, `workqueue.DefaultTypedControllerRateLimiter()`, is a `MaxOf` of an
exponential per-item limiter (5ms → 1000s) and a 10 QPS / 100 burst global
bucket: per-key backoff *and* an overall ceiling. Unless you have a specific
reason, use that.

**See also:** wq1 (`Forget`, which resets these counters) · wq3 (the cap that ends the
retries) · pods3 (`RetryOnConflict`, deliberately flat, and why) · the
[workqueue chapter](../README.md)

**References**

- `workqueue` rate limiters: https://pkg.go.dev/k8s.io/client-go/util/workqueue#TypedRateLimiter
- `DefaultTypedControllerRateLimiter`: https://pkg.go.dev/k8s.io/client-go/util/workqueue#DefaultTypedControllerRateLimiter
- `golang.org/x/time/rate`: https://pkg.go.dev/golang.org/x/time/rate
