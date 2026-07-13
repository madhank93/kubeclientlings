// setup3
//
// The zero-value rest.Config works, but production clients always tune it:
//   - QPS / Burst: client-side rate limits (default 5/10 — far too low for
//     controllers; starved clients show mysterious multi-second latency)
//   - Timeout: without it a hung API server hangs your program forever
//   - UserAgent: shows up in API server audit logs, so operators can tell
//     WHICH client is hammering the API
//
// From here on, exkit.MustRESTConfig() does the loading dance from setup1
// for you. Tune the returned config so the checks below pass, then prove it
// still works with a real round-trip.
//
// I AM NOT DONE
package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"

	"github.com/madhank93/clientlings/internal/exkit"
)

func main() {
	config := exkit.MustRESTConfig()

	// The config is completely untuned. Set QPS, Burst, Timeout and
	// UserAgent so every check below passes.

	if config.QPS < 20 {
		fmt.Printf("❌ QPS is %.0f — the default (5) starves controllers; raise it to at least 20\n", config.QPS)
		os.Exit(1)
	}
	if config.Burst < int(config.QPS) {
		fmt.Printf("❌ Burst (%d) should be at least QPS (%.0f) or bursts get throttled immediately\n", config.Burst, config.QPS)
		os.Exit(1)
	}
	if config.Timeout == 0 {
		fmt.Println("❌ Timeout is zero — a hung API server would hang this program forever")
		os.Exit(1)
	}
	if config.UserAgent == "" {
		fmt.Println("❌ UserAgent is empty — set one so audit logs can identify this client")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("❌ could not create clientset: %v\n", err)
		os.Exit(1)
	}
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		fmt.Printf("❌ round-trip failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ tuned client round-trip OK — server is Kubernetes %s\n", version.GitVersion)
	fmt.Println("\n🎉 your rest.Config is production-shaped: rate limits, timeout, identity")
}
