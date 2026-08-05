// svc2
//
// Endpoints moved to the discovery.k8s.io/v1 API: EndpointSlices. You don't
// Get them by service name — you LIST them by the well-known label
// kubernetes.io/service-name (exported as discoveryv1.LabelServiceName).
// Guessing that label wrong returns an empty list, not an error.
//
// List the EndpointSlices that belong to the web service.
package main

import (
	"context"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	ctx, cancel, cs, ns := exkit.Begin("svc2")
	defer cancel()

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 2), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}
	if _, err := cs.CoreV1().Services(ns).Create(ctx, exkit.WebService(ns, "web"), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating service: %v", err)
	}

	// The endpoint slice controller needs ready pods before it fills slices.
	exkit.WaitFor(ctx, "2 ready endpoints behind the service", func(ctx context.Context) (bool, error) {
		// Slices are named web-x7k2p, not "web", so there is nothing to Get.
		// The constant is "kubernetes.io/service-name" — using it rather than
		// a typed string is the point: a wrong label is an empty list, not an
		// error, and spins until the timeout.
		slices, err := cs.DiscoveryV1().EndpointSlices(ns).List(ctx, metav1.ListOptions{
			LabelSelector: discoveryv1.LabelServiceName + "=web",
		})
		if err != nil {
			return false, err
		}
		// One Service can have several slices (100 endpoints each by default),
		// hence the nested loop.
		ready := 0
		for _, slice := range slices.Items {
			for _, ep := range slice.Endpoints {
				// *bool: nil means "unknown", which is not false.
				if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
					ready++
				}
			}
		}
		return ready == 2, nil
	})

	exkit.Successf("found the service's EndpointSlices via %s — 2 ready backends", discoveryv1.LabelServiceName)
}
