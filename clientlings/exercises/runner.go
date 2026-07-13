package exercises

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// DefaultTimeout bounds a single exercise run. Exercises talk to a live kind
// cluster, so runs are slow but must never hang the watch loop forever.
// Override per exercise with `timeout = <seconds>` in info.toml.
const DefaultTimeout = 120 * time.Second

type Result struct {
	Exercise Exercise
	Out      string
	Err      string
}

func (e Exercise) timeout() time.Duration {
	if e.Timeout > 0 {
		return time.Duration(e.Timeout) * time.Second
	}
	return DefaultTimeout
}

func (e Exercise) Run() (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout())
	defer cancel()

	args := BuildArgs(e)
	cmd := exec.CommandContext(ctx, "go", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintf(&stderr,
			"\nexercise timed out after %s — is the cluster up? Try `clientlings doctor`.\n",
			e.timeout())
		if err == nil {
			err = ctx.Err()
		}
	}

	return Result{Exercise: e, Out: stdout.String(), Err: stderr.String()}, err
}

func BuildArgs(e Exercise) []string {
	args := []string{}
	if e.Mode == "compile" {
		args = append(args, "run")
	} else {
		args = append(args, "test", "-v", "-race")
	}

	args = append(args, fmt.Sprintf("./%s", e.Path))
	return args
}
