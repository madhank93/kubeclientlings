# workqueue

The rate-limiting machinery that keeps a controller sane under failure: the
Done-vs-Forget distinction (release processing vs reset backoff), exponential
backoff delays that double on every failure, and the handleErr retry cap that
drops a poison item instead of requeuing it forever.

These exercises are pure queue mechanics — no cluster required.

## Resources

- [workqueue package](https://pkg.go.dev/k8s.io/client-go/util/workqueue)
- [RateLimitingInterface](https://pkg.go.dev/k8s.io/client-go/util/workqueue#TypedRateLimitingInterface)
- [Writing controllers — worker & handleErr](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
- [sample-controller handleErr](https://github.com/kubernetes/sample-controller/blob/master/controller.go)
