// Package preflight checks that the local toolchain and kind cluster are
// ready and returns actionable issues for the CLI/TUI to surface.
// Ported from kubelings' internal/preflight and extended with cluster checks.
package preflight

import (
	"os/exec"

	"github.com/madhank93/kubeclientlings/kubeclientlings/cluster"
)

// Issue is a missing prerequisite plus how to fix it.
type Issue struct {
	Msg string
	Fix string
}

// Check verifies required binaries exist, Docker is running, the kind
// cluster exists, and its API server answers. Ordered cheapest-first;
// later checks are skipped when their prerequisites already failed.
func Check() []Issue {
	var issues []Issue
	fixes := map[string]string{
		"docker":  "install a Docker runtime (OrbStack or Docker Desktop)",
		"kind":    "mise install (or: brew install kind)",
		"kubectl": "mise install (or: brew install kubernetes-cli)",
		"go":      "mise install (or: brew install go)",
	}
	// Stable order for predictable banners.
	for _, b := range []string{"docker", "kind", "kubectl", "go"} {
		if _, err := exec.LookPath(b); err != nil {
			issues = append(issues, Issue{Msg: b + " not found", Fix: fixes[b]})
		}
	}
	if len(issues) > 0 {
		return issues
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		issues = append(issues, Issue{
			Msg: "Docker runtime not running",
			Fix: "start OrbStack (or Docker), then retry",
		})
		return issues
	}

	if !cluster.Exists() {
		issues = append(issues, Issue{
			Msg: "kind cluster \"" + cluster.Name + "\" not found",
			Fix: "clientlings up",
		})
		return issues
	}

	// API server answering? Cheap readiness probe with a tight timeout.
	if err := exec.Command("kubectl", "--context", cluster.Context,
		"get", "--raw", "/readyz", "--request-timeout=5s").Run(); err != nil {
		issues = append(issues, Issue{
			Msg: "cluster API server not answering",
			Fix: "clientlings down && clientlings up",
		})
	}

	return issues
}
