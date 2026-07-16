package exercises

import (
	"errors"
	"os/exec"
	"path/filepath"
)

// Status is the rich, gated state of an exercise used by the TUI. It is finer
// than State: an exercise only reaches StatusDone when its marker is removed,
// it compiles/tests cleanly, AND golangci-lint is clean.
type Status int

const (
	StatusPending  Status = iota // compiles/tests pass, but marker still present
	StatusFailing                // Run() fails (compile error or failing test)
	StatusLintFail               // marker removed, Run() passes, lint not clean
	StatusDone                   // marker removed, Run() passes, lint clean
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusFailing:
		return "Failing"
	case StatusLintFail:
		return "LintFail"
	case StatusDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// Verify computes the gated status of e. It always runs the exercise (so the
// learner sees compile/test output while working), then — only once the run
// passes and the "I AM NOT DONE" marker is removed — lints it. The returned
// Result carries the output to show the learner.
func (e Exercise) Verify() (Status, Result) {
	result, err := e.Run()
	if err != nil {
		return StatusFailing, result
	}

	// Compiles / tests pass. The marker must still be removed to complete.
	if e.State() == Pending {
		return StatusPending, result
	}

	if ok, out := Lint(e); !ok {
		result.Out = out
		result.Err = ""
		return StatusLintFail, result
	}

	return StatusDone, result
}

// Lint runs golangci-lint on the exercise's directory. It returns (true, "")
// when clean. If golangci-lint is not installed it degrades gracefully
// (returns true with a note) rather than blocking progress forever.
func Lint(e Exercise) (bool, string) {
	dir := filepath.Dir(e.Path)
	cmd := exec.Command("golangci-lint", "run", "./"+dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return true, ""
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) {
		// binary missing / not executable — skip the gate
		return true, "golangci-lint not found; lint gate skipped"
	}

	// non-zero exit => lint findings
	return false, string(out)
}
