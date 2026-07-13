package exercises

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// PathRoot is the only directory tree an exercise path may point into. It is a
// var (not a const) so tests, which load fixture info.toml files laid out
// differently, can disable confinement by setting it to "". Production code
// leaves it at the default.
var PathRoot = "exercises"

// validatePath rejects exercise paths that could escape the exercises tree.
//
// info.toml is mostly trusted, but a malformed or hostile entry must never let
// `go run`, golangci-lint, the editor, or reset touch files outside the repo's
// exercises/ directory. Validating once, centrally, at load time (rather than
// in the runner alone) means every consumer of Exercise.Path is protected and
// a bad entry fails fast with a clear message instead of at execution.
//
// A path is accepted only when it is non-empty, relative, already canonical
// (so "a/../b" and "./a" style traversal tricks are refused outright), and
// rooted at PathRoot.
func validatePath(p string) error {
	if p == "" {
		return fmt.Errorf("exercise path is empty")
	}
	if filepath.IsAbs(p) || path.IsAbs(p) || strings.HasPrefix(p, "\\") {
		return fmt.Errorf("exercise path must be relative: %q", p)
	}

	// Normalize to forward slashes so the rules behave identically on Windows.
	slash := filepath.ToSlash(p)
	if slash != path.Clean(slash) {
		return fmt.Errorf("exercise path is not canonical (possible traversal): %q", p)
	}
	if PathRoot != "" && slash != PathRoot && !strings.HasPrefix(slash, PathRoot+"/") {
		return fmt.Errorf("exercise path escapes %s/: %q", PathRoot, p)
	}
	return nil
}
