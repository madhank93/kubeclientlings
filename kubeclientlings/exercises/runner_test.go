package exercises

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// hangFixture is an exercise that never terminates. Before RunContext existed,
// selecting it in the TUI wedged the whole program with no way out.
var hangFixture = Exercise{
	Name: "hang1",
	Path: "../fixtures/hang1/main.go",
	Mode: "compile",
}

func TestRunContextHonoursDeadline(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	// Generous enough to cover compiling the fixture, short enough that a
	// regression fails the test rather than hanging it.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	done := make(chan Result, 1)
	go func() {
		res, _ := hangFixture.RunContext(ctx)
		done <- res
	}()

	select {
	case res := <-done:
		if !strings.Contains(res.Err, "cancelled") && !strings.Contains(res.Err, "timed out") {
			t.Errorf("run ended without a cancellation note; stderr = %q", res.Err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("RunContext ignored its deadline")
	}
}

func TestRunContextCancelReturnsPromptly(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_, _ = hangFixture.RunContext(ctx)
		close(done)
	}()

	// Give it long enough to compile and start looping, then pull the plug.
	time.Sleep(8 * time.Second)
	cancel()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("cancel did not stop the run: the process group outlived it")
	}
}

func TestCappedBufferTruncates(t *testing.T) {
	var b cappedBuffer
	chunk := strings.Repeat("x", 4096)
	for written := 0; written < maxOutput+8192; written += len(chunk) {
		n, err := b.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		// A short write would make exec treat the stream as a broken pipe and
		// fail the run for the wrong reason.
		if n != len(chunk) {
			t.Fatalf("Write returned %d, want %d", n, len(chunk))
		}
	}
	out := b.String()
	if !strings.Contains(out, "output truncated") {
		t.Error("oversized output was not marked as truncated")
	}
	if len(out) > maxOutput+256 {
		t.Errorf("retained %d bytes, want at most maxOutput plus the notice", len(out))
	}
}

func TestCappedBufferPassesSmallOutputThrough(t *testing.T) {
	var b cappedBuffer
	if _, err := b.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := b.String(); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}
