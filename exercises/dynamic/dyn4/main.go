// dyn4
//
// How does kubectl know what resources exist? It ASKS — the discovery API
// lists every group/version the server serves and every resource in them.
// But you must ask for a groupVersion that exists: apps/v1beta1 died in
// Kubernetes 1.16, and asking for it is an error, not an empty list.
//
// Discover where deployments live today.
//
// I AM NOT DONE
package main

import (
	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	_, cancel, _, _ := exkit.Begin("dyn4")
	defer cancel()

	disco := exkit.MustDiscovery()

	// apps/v1beta1 was removed in Kubernetes 1.16 — the server hasn't
	// served it in years, and discovery says so with an error. Where do
	// deployments live now?
	resources, err := disco.ServerResourcesForGroupVersion("apps/v1beta1")
	if err != nil {
		exkit.Failf("asking the server about the group/version: %v", err)
	}

	found := false
	for _, r := range resources.APIResources {
		if r.Name == "deployments" {
			found = true
			exkit.AssertEqual("deployments Kind", r.Kind, "Deployment")
			exkit.AssertTrue("deployments are namespaced", r.Namespaced)
		}
	}
	exkit.AssertTrue("deployments found in the discovered group/version", found)
	exkit.Successf("discovery is how clients map names to endpoints — never hardcode dead groupVersions")
}
