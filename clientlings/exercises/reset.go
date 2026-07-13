package exercises

import (
	"fmt"
	"os/exec"
)

// Reset restores an exercise file to its committed (shipped, broken) state with
// git. Requires the repo to be a git checkout and the file to be committed.
func Reset(e Exercise) error {
	cmd := exec.Command("git", "checkout", "HEAD", "--", e.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed for %s: %w: %s", e.Path, err, out)
	}
	return nil
}
