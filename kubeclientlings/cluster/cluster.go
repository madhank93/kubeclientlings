// Package cluster manages the local kind cluster every exercise runs against.
// Ported from kubelings' run-challenge-local.sh up/down verbs so the logic
// ships inside the single clientlings binary.
package cluster

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	// Name is the kind cluster name; the kubeconfig context is "kind-" + Name.
	Name    = "kubeclientlings"
	Context = "kind-" + Name

	// Workers is the number of worker nodes (plus one control-plane).
	Workers = 2
)

// Exists reports whether the kind cluster is already created.
func Exists() bool {
	out, err := exec.Command("kind", "get", "clusters").Output()
	if err != nil {
		return false
	}
	return slices.Contains(strings.Split(strings.TrimSpace(string(out)), "\n"), Name)
}

// Up creates the cluster (1 control-plane + Workers workers), switches the
// kubeconfig context to it, and waits for all nodes to be Ready. It is
// idempotent: an existing cluster is left untouched.
func Up() error {
	if Exists() {
		fmt.Printf("cluster %q already exists.\n", Name)
	} else {
		cfg, err := os.CreateTemp("", "clientlings-kind-*.yaml")
		if err != nil {
			return err
		}
		defer os.Remove(cfg.Name())

		var b strings.Builder
		b.WriteString("kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n  - role: control-plane\n")
		for range Workers {
			b.WriteString("  - role: worker\n")
		}
		if _, err := cfg.WriteString(b.String()); err != nil {
			return err
		}
		if err := cfg.Close(); err != nil {
			return err
		}

		if err := run("kind", "create", "cluster", "--name", Name, "--config", cfg.Name()); err != nil {
			return fmt.Errorf("kind create cluster failed: %w", err)
		}
	}

	if err := run("kubectl", "config", "use-context", Context); err != nil {
		return fmt.Errorf("could not switch kubeconfig context to %s: %w", Context, err)
	}
	if err := run("kubectl", "--context", Context, "wait", "--for=condition=Ready", "nodes", "--all", "--timeout=120s"); err != nil {
		return fmt.Errorf("nodes did not become Ready: %w", err)
	}

	fmt.Printf("cluster %q ready (%d nodes). Next: clientlings watch\n", Name, Workers+1)
	return nil
}

// Down deletes the cluster.
func Down() error {
	return run("kind", "delete", "cluster", "--name", Name)
}

// run executes a command streaming its output to the terminal, so the learner
// sees kind/kubectl progress live.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
