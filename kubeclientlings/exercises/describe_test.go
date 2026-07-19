package exercises

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "main_test.go")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDescription(t *testing.T) {
	path := writeTmp(t, "// errors2\n// Wrapping an error with %w keeps the original reachable.\n// Make lookup wrap it.\n\n// I AM NOT DONE\npackage main_test\n")
	e := Exercise{Name: "errors2", Path: path}
	want := "Wrapping an error with %w keeps the original reachable. Make lookup wrap it."
	if got := e.Description(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDescriptionFullHeader(t *testing.T) {
	// Wrapped lines join with a space; a blank comment line is a paragraph
	// break; the name header and the NOT DONE marker never leak in.
	path := writeTmp(t, "// pods9\n//\n// First paragraph explains the concept\n// across two wrapped lines.\n//\n// Second paragraph states the task.\n//\n// I AM NOT DONE\npackage main_test\n")
	e := Exercise{Name: "pods9", Path: path}
	want := "First paragraph explains the concept across two wrapped lines.\n\nSecond paragraph states the task."
	if got := e.Description(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDescriptionGenericSkipped(t *testing.T) {
	// stock-style header with only boilerplate -> no useful description
	path := writeTmp(t, "// variables2\n// Make me compile!\n\n// I AM NOT DONE\npackage main\n")
	e := Exercise{Name: "variables2", Path: path}
	if got := e.Description(); got != "" {
		t.Errorf("expected empty description, got %q", got)
	}
}
