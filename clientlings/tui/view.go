package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madhank93/clientlings/clientlings/exercises"
)

// Adaptive colors so the UI reads well on both light and dark terminals.
var (
	cBlue  = lipgloss.AdaptiveColor{Light: "#0550ae", Dark: "#79c0ff"}
	cLink  = lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"}
	cGreen = lipgloss.AdaptiveColor{Light: "#1a7f37", Dark: "#3fb950"}
	cRed   = lipgloss.AdaptiveColor{Light: "#cf222e", Dark: "#f85149"}
	cAmber = lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#d29922"}
	cTeal  = lipgloss.AdaptiveColor{Light: "#0e7490", Dark: "#39c5cf"}
	cDim   = lipgloss.AdaptiveColor{Light: "#6e7781", Dark: "#8b949e"}
)

var (
	topicStyle    = lipgloss.NewStyle().Bold(true).Foreground(cBlue)
	selectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true) // inverts fg/bg — readable on any theme
	doneStyle     = lipgloss.NewStyle().Foreground(cGreen)
	failStyle     = lipgloss.NewStyle().Foreground(cRed)
	lintStyle     = lipgloss.NewStyle().Foreground(cAmber)
	passStyle     = lipgloss.NewStyle().Foreground(cTeal)
	dimStyle      = lipgloss.NewStyle().Foreground(cDim)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	noticeStyle   = lipgloss.NewStyle().Foreground(cAmber)
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

	// section header styles (labelled, colored)
	secDescStyle = lipgloss.NewStyle().Bold(true).Foreground(cBlue)
	secErrStyle  = lipgloss.NewStyle().Bold(true).Foreground(cRed)
	secOkStyle   = lipgloss.NewStyle().Bold(true).Foreground(cGreen)
	secWarnStyle = lipgloss.NewStyle().Bold(true).Foreground(cAmber)
	secHintStyle = lipgloss.NewStyle().Bold(true).Foreground(cTeal)
	secOutStyle  = lipgloss.NewStyle().Bold(true).Foreground(cDim)
	searchStyle  = lipgloss.NewStyle().Bold(true).Foreground(cAmber)
)

// topicOf extracts the topic directory from an exercise path.
func topicOf(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return path
}

func (m *Model) layout() {
	headerH, footerH := 3, 2
	bodyH := m.height - headerH - footerH
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := m.width * 4 / 10
	if leftW < 28 {
		leftW = 28
	}
	rightW := m.width - leftW - 6 // borders + padding
	if rightW < 20 {
		rightW = 20
	}
	m.output.Width = rightW
	m.output.Height = bodyH - 3 // leave a title line inside the right pane
	// leave room on the header line for the title + "n/N (p%)  🔥 streak N"
	m.progress.Width = max(10, m.width-46)
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	if m.phase == phaseWelcome {
		return m.welcome()
	}
	return strings.Join([]string{
		m.header(),
		m.body(),
		m.footer(),
	}, "\n")
}

var (
	linkStyle  = lipgloss.NewStyle().Foreground(cLink).Underline(true)
	labelStyle = lipgloss.NewStyle().Foreground(cDim).Width(12)
	markStyle  = lipgloss.NewStyle().Foreground(cAmber)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(cTeal).Padding(1, 3)
)

// welcome renders the splash screen shown before the main view.
func (m Model) welcome() string {
	title := titleStyle.Foreground(cTeal).Render("🐹  clientlings")
	tagline := dimStyle.Render("Learn Go the rustlings way — 112 exercises, basics → advanced")

	meta := lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Repo")+linkStyle.Render("https://github.com/madhank93/clientlings"),
		labelStyle.Render("Site")+linkStyle.Render("https://clientlings.madhan.app"),
		labelStyle.Render("Maintainer")+"Madhan Kumaravelu  "+dimStyle.Render("(@madhank93)"),
	)

	how := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("How it works"),
		"  • Open the highlighted exercise's file and make it compile/pass",
		"  • Remove the  "+markStyle.Render("// I AM NOT DONE")+"  marker when you think it's done",
		"  • Save — clientlings auto-runs it; "+titleStyle.Render("tests AND golangci-lint")+" must pass",
		"  • Press n to move to the next exercise",
	)

	keys := dimStyle.Render("Keys   ↑↓/jk move · ⏎ run · e edit · h hint · r reset · n next · q quit")
	cta := markStyle.Render("press any key to start →")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, tagline, "", meta, "", how, "", keys, "", cta,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(content))
}

func (m Model) header() string {
	done := m.tracker.DoneCount()
	ratio := 0.0
	if m.total > 0 {
		ratio = float64(done) / float64(m.total)
	}
	bar := m.progress.ViewAs(ratio)
	stats := fmt.Sprintf(" %d/%d (%.0f%%)   🔥 streak %d", done, m.total, ratio*100, m.tracker.Streak())
	return titleStyle.Render("clientlings") + "  " + bar + stats
}

func (m Model) body() string {
	headerH, footerH := 3, 2
	bodyH := m.height - headerH - footerH
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := m.width * 4 / 10
	if leftW < 28 {
		leftW = 28
	}

	left := paneStyle.Width(leftW).Height(bodyH).Render(m.list(bodyH))
	right := paneStyle.Width(m.output.Width).Height(bodyH).Render(m.rightPane())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// list renders the windowed topic/exercise list around the cursor. While
// filtering it collapses to a flat, query-narrowed list with a search prompt.
func (m Model) list(height int) string {
	if m.filtering {
		return m.filteredList(height)
	}

	var lines []string
	cursorLine := 0
	for i, it := range m.items {
		if it.isHeader {
			lines = append(lines, topicStyle.Render(fmt.Sprintf("%s (%d/%d)", it.topic, it.done, it.total)))
			continue
		}
		row := "  " + m.glyph(it.ex) + " " + it.ex.Name
		if i == m.cursor {
			cursorLine = len(lines)
			row = selectedStyle.Render("→ " + m.glyph(it.ex) + " " + it.ex.Name)
		}
		lines = append(lines, row)
	}
	return windowLines(lines, cursorLine, height)
}

// filteredList renders the search prompt plus only the matching exercises,
// each prefixed with its topic so flattened rows stay unambiguous.
func (m Model) filteredList(height int) string {
	lines := []string{searchStyle.Render("/") + m.filter + "▌", ""}
	cursorLine := 0
	matches := 0
	for i, it := range m.items {
		if it.isHeader || !m.matchesFilter(it.ex) {
			continue
		}
		matches++
		if i == m.cursor {
			cursorLine = len(lines)
			lines = append(lines, selectedStyle.Render("→ "+m.glyph(it.ex)+" "+it.ex.Name))
			continue
		}
		label := dimStyle.Render(topicOf(it.ex.Path)+"/") + it.ex.Name
		lines = append(lines, "  "+m.glyph(it.ex)+" "+label)
	}
	if matches == 0 {
		lines = append(lines, dimStyle.Render("  no matches"))
	}
	return windowLines(lines, cursorLine, height)
}

// glyph returns the colored status marker for an exercise row (cheap, no run).
func (m Model) glyph(ex exercises.Exercise) string {
	switch {
	case m.tracker.IsDone(ex.Name):
		return doneStyle.Render("✓")
	case ex.State() == exercises.Done: // marker removed but not verified done
		return lintStyle.Render("●")
	default:
		return dimStyle.Render("○")
	}
}

func (m Model) rightPane() string {
	ex := m.current()
	title := titleStyle.Render(ex.Name) + dimStyle.Render(" ["+ex.Mode+"] ") + m.badge()
	notice := ""
	if m.notice != "" {
		notice = "\n" + noticeStyle.Render(m.notice)
	}
	return title + notice + "\n" + m.output.View()
}

func (m Model) badge() string {
	if m.verifying {
		return m.spinner.View() + " running"
	}
	if !m.hasResult {
		return dimStyle.Render("not run")
	}
	switch m.status {
	case exercises.StatusDone:
		return doneStyle.Render("✓ done")
	case exercises.StatusLintFail:
		return lintStyle.Render("⚠ lint")
	case exercises.StatusFailing:
		return failStyle.Render("● failing")
	case exercises.StatusPending:
		return passStyle.Render("◐ remove marker")
	default:
		return dimStyle.Render("○ pending")
	}
}

// section renders a colored "Label:" header followed by its body.
func section(header, body string) string {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return header
	}
	return header + "\n" + body
}

// detail is the scrollable content of the right pane: a Description block plus
// a labelled Hint / Error / Output block, all soft-wrapped (never truncated).
func (m Model) detail() string {
	var parts []string

	if d := m.current().Description(); d != "" {
		parts = append(parts, section(secDescStyle.Render("Description:"), d))
	}

	if m.showHint {
		parts = append(parts, section(secHintStyle.Render("Hint:"), m.current().Hint))
		return strings.Join(parts, "\n\n")
	}

	switch {
	case !m.hasResult:
		parts = append(parts, dimStyle.Render("Press ⏎ to run this exercise (or just edit the file and save)."))
	case m.status == exercises.StatusFailing:
		parts = append(parts, section(secErrStyle.Render("Status:"), "Doesn't compile or a test failed."))
		if e := strings.TrimSpace(m.result.Err); e != "" {
			parts = append(parts, section(secErrStyle.Render("Error:"), e))
		}
	case m.status == exercises.StatusPending:
		parts = append(parts, section(secOkStyle.Render("Status:"), "Compiles and tests pass — remove the '// I AM NOT DONE' marker to complete it."))
	case m.status == exercises.StatusLintFail:
		parts = append(parts, section(secWarnStyle.Render("Status:"), "Lint issues — fix them to complete the exercise."))
	case m.status == exercises.StatusDone:
		parts = append(parts, section(secOkStyle.Render("Status:"), "Passed and lint-clean! Press n for the next exercise."))
	}

	// Program / tool output as its own labelled section, never glued to the status.
	if m.hasResult {
		if out := strings.TrimSpace(m.result.Out); out != "" {
			parts = append(parts, section(secOutStyle.Render("Output:"), out))
		}
	}

	return strings.Join(parts, "\n\n")
}

func (m Model) footer() string {
	return m.help.View(m.keys)
}

// refreshOutput sets the right-pane viewport content, soft-wrapped to its width
// so long compiler/test output isn't clipped.
func (m *Model) refreshOutput() {
	w := m.output.Width
	if w < 1 {
		w = 80
	}
	m.output.SetContent(lipgloss.NewStyle().Width(w).Render(m.detail()))
}

// windowLines returns at most height lines centered on the focus line.
func windowLines(lines []string, focus, height int) string {
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}
	start := focus - height/2
	if start < 0 {
		start = 0
	}
	if start+height > len(lines) {
		start = len(lines) - height
	}
	return strings.Join(lines[start:start+height], "\n")
}
