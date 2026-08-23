package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
)

// notesModel points at real exercises, so Notes() and Chapter() return content.
func notesModel(t *testing.T) Model {
	t.Helper()
	tr, _ := exercises.LoadState(filepath.Join(t.TempDir(), "s.json"))
	exs := []exercises.Exercise{
		{Name: "pods1", Path: "../../exercises/pods/pods1/main.go", Mode: "compile", Hint: "namespace"},
		{Name: "pods2", Path: "../../exercises/pods/pods2/main.go", Mode: "compile", Hint: "selector"},
	}
	m := Model{
		tracker: tr, phase: phaseMain, keys: defaultKeys(), help: help.New(),
		progress: progress.New(), spinner: spinner.New(), output: viewport.New(0, 0),
		total: len(exs),
	}
	m.items = buildItems(exs, tr)
	m.cursor = m.firstSelectable()
	return m
}

func press(t *testing.T, m Model, r rune) Model {
	t.Helper()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return nm.(Model)
}

// The chapter is one document per topic. Showing it with every exercise's
// walk-through would repeat the same pages under all seven pods exercises, so
// it lives behind its own key.
func TestChapterIsNotPartOfTheWalkthrough(t *testing.T) {
	m := notesModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)

	if m.current().Notes() == "" || m.current().Chapter() == "" {
		t.Fatal("test needs an exercise with both notes and a chapter")
	}

	m = press(t, m, 'x')
	if !m.showNotes || m.showChapter {
		t.Fatalf("x should open the note alone: notes=%v chapter=%v", m.showNotes, m.showChapter)
	}
	withNote := m.output.TotalLineCount()

	m = press(t, m, 'c')
	if !m.showChapter {
		t.Fatal("c should open the chapter")
	}
	if m.output.TotalLineCount() <= withNote {
		t.Error("opening the chapter added nothing to the pane")
	}
	if m.output.YOffset != m.chapterTop {
		t.Errorf("opening the chapter should scroll to it: want %d, got %d", m.chapterTop, m.output.YOffset)
	}

	m = press(t, m, 'c')
	if m.showChapter {
		t.Error("c should toggle the chapter back off")
	}

	// Moving to another exercise starts from a clean slate.
	m = press(t, m, 'c')
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	if m.showChapter || m.showNotes {
		t.Errorf("selection change should reset the panes: notes=%v chapter=%v", m.showNotes, m.showChapter)
	}
}

// The key line in the header is hand-written, so it drifts from the bindings.
func TestHeaderKeyLineMentionsBothKeys(t *testing.T) {
	m := notesModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = nm.(Model)
	v := m.View()
	for _, want := range []string{"x explain", "c chapter"} {
		if !strings.Contains(v, want) {
			t.Errorf("header key line is missing %q", want)
		}
	}
}

// The Learn section is long, so a re-verify on every save must not throw a
// reader back to where the section starts.
func TestReverifyKeepsScrollPosition(t *testing.T) {
	m := notesModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)

	m = press(t, m, 'x')
	if m.output.YOffset != m.notesTop {
		t.Fatalf("opening notes should scroll to the Learn section: want %d, got %d", m.notesTop, m.output.YOffset)
	}

	m.output.SetYOffset(m.notesTop + 5)
	scrolled := m.output.YOffset

	nm, _ = m.Update(verifiedMsg{name: "pods1", status: exercises.StatusDone, result: exercises.Result{Out: "ok"}})
	m = nm.(Model)

	if m.output.YOffset < scrolled {
		t.Errorf("re-verify yanked the viewport back: was %d, now %d", scrolled, m.output.YOffset)
	}
}

// The terminal reads the ascii twin of each diagram; the mermaid source is for
// the docs site and must never reach the pane.
func TestChapterDropsMermaidSource(t *testing.T) {
	m := notesModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)

	m = press(t, m, 'c')
	if strings.Contains(m.output.View(), "```mermaid") {
		t.Error("mermaid fence leaked into the terminal pane")
	}
}
