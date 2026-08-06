package exercises

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var notDoneRegex = regexp.MustCompile(`(?m)^\s*///?\s*I\s+AM\s+NOT\s+DONE`)

type Exercise struct {
	Name    string
	Path    string
	Mode    string
	Hint    string
	Desc    string
	Timeout int // seconds; 0 means DefaultTimeout
}

// Notes returns the exercise's teaching walk-through: the contents of a
// notes.md file sitting next to the exercise, or "" when absent. Shown once the
// learner solves the exercise (or on demand via the Explain key) to explain the
// client-go machinery the exercise exercised.
func (e Exercise) Notes() string {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(e.Path), "notes.md"))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n")
}

func (e Exercise) State() State {
	data, err := os.ReadFile(e.Path)
	if err != nil {
		return Pending
	}

	if notDoneRegex.Match(data) {
		return Pending
	}
	return Done
}

type State int

const (
	Pending State = iota + 1
	Done
)

func (s State) String() string {
	return [...]string{"Pending", "Done"}[s-1]
}
