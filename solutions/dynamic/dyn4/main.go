// dyn4
//
// How does kubectl know what resources exist? It ASKS — the discovery API
// lists every group/version the server serves and every resource in them.
// But you must ask for a groupVersion that exists: apps/v1beta1 died in
// Kubernetes 1.16, and asking for it is an error, not an empty list.
//
// Discover where deployments live today.
package main

import (
	"github.com/madhank93/kubeclientlings/internal/exkit"
)

func main() {
	_, cancel, _, _ := exkit.Begin("dyn4")
	defer cancel()

	disco := exkit.MustDiscovery()

	// Deployments have lived in apps/v1 since Kubernetes 1.9.
	resources, err := disco.ServerResourcesForGroupVersion("apps/v1")
	if err != nil {
		exkit.Failf("asking the server about apps/v1: %v", err)
	}

	found := false
	for _, r := range resources.APIResources {
		if r.Name == "deployments" {
			found = true
			exkit.AssertEqual("deployments Kind", r.Kind, "Deployment")
			exkit.AssertTrue("deployments are namespaced", r.Namespaced)
		}
	}
	exkit.AssertTrue("deployments found in apps/v1", found)
	exkit.Successf("discovery is how clients map names to endpoints — never hardcode dead groupVersions")
}
