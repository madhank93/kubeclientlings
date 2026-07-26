package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWatchRelevant(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"exercises/variables/variables1/main.go", true},
		{"exercises/errors/errors1/main_test.go", true},
		{"go.mod", true},
		{"go.sum", true},
		{"README.md", false},
		{"exercises/.DS_Store", false},
		{"exercises/variables/.main.go.swp", false},
		{"exercises/variables/variables1", false},
	}
	for _, c := range cases {
		if got := watchRelevant(c.path); got != c.want {
			t.Errorf("watchRelevant(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// A burst of writes — what an editor actually does on save — must collapse
// into a single message, or every save re-runs the exercise several times.
func TestWatcherDebouncesBurst(t *testing.T) {
	dir := t.TempDir()
	ch := make(chan tea.Msg, 16)
	w, err := startWatcher(dir, ch)
	if err != nil {
		t.Fatalf("startWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	file := filepath.Join(dir, "main.go")
	for range 5 {
		if err := os.WriteFile(file, []byte("package main\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("no fileChangedMsg after writes")
	}
	// Nothing further should arrive: the burst was one logical save.
	select {
	case <-ch:
		t.Fatal("burst produced more than one fileChangedMsg")
	case <-time.After(2 * debounce):
	}
}

// Directories created after startup must be watched too — fsnotify watches are
// per-directory and do not recurse.
func TestWatcherPicksUpNewDirs(t *testing.T) {
	dir := t.TempDir()
	ch := make(chan tea.Msg, 16)
	w, err := startWatcher(dir, ch)
	if err != nil {
		t.Fatalf("startWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	sub := filepath.Join(dir, "newtopic", "ex1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Let the Create events land and register the new dirs.
	time.Sleep(200 * time.Millisecond)
	drain(ch)

	if err := os.WriteFile(filepath.Join(sub, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("write in a directory created after startup went unnoticed")
	}
}

func drain(ch chan tea.Msg) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
