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
	if got := e.Description(); got != "Wrapping an error with %w keeps the original reachable." {
		t.Errorf("got %q", got)
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
