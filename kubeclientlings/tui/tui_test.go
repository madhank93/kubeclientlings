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

func TestTopicOf(t *testing.T) {
	if got := topicOf("exercises/errors/errors1/main_test.go"); got != "errors" {
		t.Errorf("got %q, want errors", got)
	}
}

func TestWindowLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	out := windowLines(lines, 4, 3)
	if n := len(strings.Split(out, "\n")); n != 3 {
		t.Errorf("want 3 windowed lines, got %d", n)
	}
	if !strings.Contains(out, "e") {
		t.Errorf("window around focus should include 'e': %q", out)
	}
}

func TestBuildItemsGroups(t *testing.T) {
	tr, _ := exercises.LoadState(filepath.Join(t.TempDir(), "s.json"))
	exs := []exercises.Exercise{
		{Name: "v1", Path: "exercises/variables/v1/main.go"},
		{Name: "v2", Path: "exercises/variables/v2/main.go"},
		{Name: "f1", Path: "exercises/functions/f1/main.go"},
	}
	items := buildItems(exs, tr)
	// 2 headers + 3 exercises
	if len(items) != 5 {
		t.Fatalf("want 5 items, got %d", len(items))
	}
	if !items[0].isHeader || items[0].topic != "variables" || items[0].total != 2 {
		t.Errorf("bad variables header: %+v", items[0])
	}
}

// build a minimal model without touching the filesystem watcher
func testModel(t *testing.T) Model {
	t.Helper()
	tr, _ := exercises.LoadState(filepath.Join(t.TempDir(), "s.json"))
	exs := []exercises.Exercise{
		{Name: "v1", Path: "exercises/variables/v1/main.go", Mode: "compile", Hint: "do it"},
		{Name: "f1", Path: "exercises/functions/f1/main.go", Mode: "test", Hint: "fix it"},
	}
	m := Model{
		tracker:  tr,
		phase:    phaseMain,
		keys:     defaultKeys(),
		help:     help.New(),
		progress: progress.New(),
		spinner:  spinner.New(),
		output:   viewport.New(0, 0),
		total:    len(exs),
	}
	m.items = buildItems(exs, tr)
	m.cursor = m.firstSelectable()
	return m
}

func TestVerifiedDoneMarksAndStays(t *testing.T) {
	m := testModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)

	start := m.current().Name // v1
	nm, _ = m.Update(verifiedMsg{name: start, status: exercises.StatusDone, result: exercises.Result{Out: "ok"}})
	m = nm.(Model)

	if !m.tracker.IsDone(start) {
		t.Errorf("expected %s marked done in tracker", start)
	}
	// stays on the solved exercise so the result is visible; user presses n
	if m.current().Name != start {
		t.Errorf("expected cursor to stay on %s, got %s", start, m.current().Name)
	}
	if m.status != exercises.StatusDone {
		t.Errorf("expected status Done to remain shown")
	}
}

func TestWelcomeDismiss(t *testing.T) {
	m := testModel(t)
	m.phase = phaseWelcome
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = nm.(Model)

	if !strings.Contains(m.View(), "KubeClientlings") || !strings.Contains(m.View(), "press any key") {
		t.Error("welcome screen missing expected content")
	}
	// any non-quit key dismisses to the main screen
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = nm.(Model)
	if m.phase != phaseMain {
		t.Error("expected welcome to dismiss to main screen")
	}
}

func TestViewAndNavigationNoPanic(t *testing.T) {
	m := testModel(t)

	// size the layout
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)
	if m.View() == "" {
		t.Fatal("expected non-empty view")
	}

	// move down, toggle hint — must not panic and must keep rendering
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = nm.(Model)
	if !m.showHint {
		t.Error("expected hint toggled on")
	}
	if !strings.Contains(m.View(), "KubeClientlings") {
		t.Error("header missing from view")
	}
}
