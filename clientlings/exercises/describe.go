package exercises

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// genericComment matches the boilerplate header lines that aren't descriptive.
var genericComment = regexp.MustCompile(`(?i)^make( me compile| the tests? pass|.*pass)`)

// mdLink turns a markdown link [text](url) into just text.
var mdLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

// Description returns a one-line summary of what the exercise is about, taken
// from the first meaningful comment in the exercise file (the line after the
// "// <name>" header). Returns "" when there's nothing useful.
func (e Exercise) Description() string {
	// An explicit desc in info.toml wins (specific, non-spoiler one-liner).
	if strings.TrimSpace(e.Desc) != "" {
		return strings.TrimSpace(e.Desc)
	}

	data, err := os.ReadFile(e.Path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			if trimmed == "" {
				continue // allow blank lines above/within the header
			}
			break // reached code; stop
		}
		c := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if c == "" || sameIdent(c, e.Name) || notDoneRegex.MatchString(line) || genericComment.MatchString(c) {
			continue
		}
		return mdLink.ReplaceAllString(c, "$1")
	}
	// Stock exercises often have only a "Make me compile!" header; fall back to
	// the topic README's first sentence so every exercise shows something.
	return topicSummary(e.Path)
}

// sameIdent reports whether two strings name the same exercise ignoring case
// and separators, so a readable title line like "// anonymous functions1" is
// recognized as the header for exercise "anonymous_functions1" (and skipped),
// while a descriptive title like "morefn1 — recursion" is not.
func sameIdent(a, b string) bool {
	norm := func(s string) string {
		var sb strings.Builder
		for _, r := range strings.ToLower(s) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				sb.WriteRune(r)
			}
		}
		return sb.String()
	}
	return norm(a) == norm(b)
}

// topicSummary returns the first sentence of the topic's README, or "".
func topicSummary(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	data, err := os.ReadFile(filepath.Join("exercises", parts[1], "README.md"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		t = mdLink.ReplaceAllString(t, "$1")
		if i := strings.Index(t, ". "); i > 0 {
			return t[:i+1]
		}
		return t
	}
	return ""
}
