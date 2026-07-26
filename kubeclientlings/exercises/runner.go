package exercises

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// DefaultTimeout bounds a single exercise run. Exercises talk to a live kind
// cluster, so runs are slow but must never hang the watch loop forever.
// Override per exercise with `timeout = <seconds>` in info.toml.
const DefaultTimeout = 120 * time.Second

// maxOutput caps how much of a run's stdout/stderr is retained. A runaway
// print loop — or a controller logging every reconcile — emits gigabytes in
// seconds, and all of it was being held in memory and handed to the renderer.
const maxOutput = 1 << 20 // 1 MiB per stream

type Result struct {
	Exercise Exercise
	Out      string
	Err      string
}

// RunTimeout is the deadline for a single run of e: its `timeout` from
// info.toml, or DefaultTimeout. Exported so the TUI can build a context with
// the same bound the runner would have applied itself.
func (e Exercise) RunTimeout() time.Duration {
	if e.Timeout > 0 {
		return time.Duration(e.Timeout) * time.Second
	}
	return DefaultTimeout
}

// Run executes the exercise under its configured timeout.
func (e Exercise) Run() (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.RunTimeout())
	defer cancel()
	return e.RunContext(ctx)
}

// RunContext executes the exercise, stopping it when ctx is done.
//
// The command runs in its own process group and cancellation signals the
// group, not just the direct child: `go test` and `go run` are supervisors
// that spawn the compiled binary. Signalling only the parent left the exercise
// itself alive — still holding its namespace, its informers and its
// port-forwards against the kind cluster — after the run had "stopped".
func (e Exercise) RunContext(ctx context.Context) (Result, error) {
	cmd := exec.CommandContext(ctx, "go", BuildArgs(e)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Negative pid addresses the whole group. SIGKILL rather than SIGTERM:
		// the point of a cancel is that it takes effect now.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	// A grandchild that inherited the stdout pipe can outlive the group kill;
	// without a bound, Wait blocks on pipe EOF forever.
	cmd.WaitDelay = 3 * time.Second

	var stdout, stderr cappedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	res := Result{Exercise: e, Out: stdout.String(), Err: stderr.String()}
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		res.Err += fmt.Sprintf(
			"\nexercise timed out after %s — is the cluster up? Try `kubeclientlings doctor`.\n",
			e.RunTimeout())
		if err == nil {
			err = ctx.Err()
		}
	case errors.Is(ctx.Err(), context.Canceled):
		res.Err += "\ncancelled.\n"
		if err == nil {
			err = ctx.Err()
		}
	}
	return res, err
}

// cappedBuffer is an io.Writer that keeps at most maxOutput bytes and then
// records that it truncated, instead of growing without bound.
type cappedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := maxOutput - c.buf.Len(); room > 0 {
		if len(p) > room {
			c.buf.Write(p[:room])
			c.truncated = true
		} else {
			c.buf.Write(p)
		}
	} else {
		c.truncated = true
	}
	// Report the full length: a short write would make exec treat the capped
	// stream as a broken pipe and fail the run for the wrong reason.
	return len(p), nil
}

func (c *cappedBuffer) String() string {
	if c.truncated {
		return fmt.Sprintf("%s\n… output truncated at %d KiB.", c.buf.String(), maxOutput/1024)
	}
	return c.buf.String()
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
