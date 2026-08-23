package exercises

import "strings"

// StripFences removes fenced code blocks tagged with the given info string.
//
// Teaching markdown carries each diagram twice: a ```mermaid block the docs
// site renders as a graph, and an ```ascii block that reads correctly in a
// terminal and on GitHub. Each renderer drops the form it cannot show.
func StripFences(md, lang string) string {
	var (
		out      []string
		fence    string // the backtick run that opened the current block, "" when outside
		skipping bool
	)
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)

		if fence == "" {
			if ticks := openingFence(trimmed); ticks != "" {
				fence = ticks
				skipping = strings.TrimSpace(strings.TrimPrefix(trimmed, ticks)) == lang
			}
		} else if trimmed == fence {
			if !skipping {
				out = append(out, line)
			}
			fence, skipping = "", false
			continue
		}

		if !skipping {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// openingFence returns the backtick run starting a fenced block, or "".
func openingFence(trimmed string) string {
	ticks := len(trimmed) - len(strings.TrimLeft(trimmed, "`"))
	if ticks < 3 {
		return ""
	}
	return trimmed[:ticks]
}
