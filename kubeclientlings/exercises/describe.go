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

// Description returns the exercise's full header comment: every meaningful
// comment line above the code, with wrapped lines joined by a space and blank
// comment lines kept as paragraph breaks. Returns "" when there's nothing
// useful.
func (e Exercise) Description() string {
	// An explicit desc in info.toml wins (specific, non-spoiler one-liner).
	if strings.TrimSpace(e.Desc) != "" {
		return strings.TrimSpace(e.Desc)
	}

	data, err := os.ReadFile(e.Path)
	if err != nil {
		return ""
	}
	var paras []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
		}
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
		if c == "" {
			flush() // blank comment line = paragraph break
			continue
		}
		if sameIdent(c, e.Name) || notDoneRegex.MatchString(line) || genericComment.MatchString(c) {
			continue
		}
		cur = append(cur, mdLink.ReplaceAllString(c, "$1"))
	}
	flush()
	if len(paras) > 0 {
		return strings.Join(paras, "\n\n")
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
