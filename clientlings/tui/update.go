package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madhank93/clientlings/clientlings/exercises"
)

// editorClosedMsg is sent when the external editor process exits.
type editorClosedMsg struct{}

// editorCommand builds the command to open path in the user's editor,
// honoring $VISUAL then $EDITOR, falling back to vi.
func editorCommand(path string) *exec.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	fields := strings.Fields(editor) // editor may include flags, e.g. "code -w"
	args := append(fields[1:], path)
	return exec.Command(fields[0], args...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.refreshOutput()
		return m, nil

	case tea.KeyMsg:
		if m.phase == phaseWelcome {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			m.phase = phaseMain
			m.refreshOutput()
			return m, nil
		}
		return m.handleKey(msg)

	case spinner.TickMsg:
		if m.verifying {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case fileChangedMsg:
		// re-verify the current exercise on any save; keep listening
		m.verifying = true
		m.notice = ""
		return m, tea.Batch(waitForChange(m.watchCh), verifyCmd(m.current()), m.spinner.Tick)

	case editorClosedMsg:
		// returned from $EDITOR — re-verify the exercise just edited
		m.verifying = true
		return m, tea.Batch(verifyCmd(m.current()), m.spinner.Tick)

	case verifiedMsg:
		return m.handleVerified(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Search):
		m.filtering = true
		m.filter = ""
		m.refreshOutput()
		return m, nil

	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case key.Matches(msg, m.keys.Run):
		m.verifying = true
		m.notice = ""
		return m, tea.Batch(verifyCmd(m.current()), m.spinner.Tick)

	case key.Matches(msg, m.keys.Edit):
		cur := m.current()
		return m, tea.ExecProcess(editorCommand(cur.Path), func(error) tea.Msg {
			return editorClosedMsg{}
		})

	case key.Matches(msg, m.keys.Hint):
		m.showHint = !m.showHint
		m.refreshOutput()
		return m, nil

	case key.Matches(msg, m.keys.Reset):
		cur := m.current()
		if err := exercises.Reset(cur); err != nil {
			m.notice = "reset failed: " + err.Error()
		} else {
			m.tracker.Unmark(cur.Name)
			_ = m.tracker.Save()
			m.refreshHeaderCounts()
			m.notice = "reset " + cur.Name + " to original"
		}
		m.hasResult = false
		m.status = exercises.StatusPending
		m.verifying = true
		return m, tea.Batch(verifyCmd(cur), m.spinner.Tick)

	case key.Matches(msg, m.keys.Next):
		m.advance()
		m.onSelectionChange()
		m.verifying = true
		return m, tea.Batch(verifyCmd(m.current()), m.spinner.Tick)
	}

	// let the viewport scroll (pgup/pgdn etc.)
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return m, cmd
}

// handleFilterKey drives the incremental search overlay in the list pane:
// type to narrow, ↑/↓ to move within matches, Enter to run the highlighted
// exercise, Esc to cancel.
func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filter = ""
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case tea.KeyEnter:
		m.filtering = false
		m.filter = ""
		m.onSelectionChange()
		m.verifying = true
		return m, tea.Batch(verifyCmd(m.current()), m.spinner.Tick)

	case tea.KeyUp:
		m.moveCursor(-1)
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case tea.KeyDown:
		m.moveCursor(1)
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case tea.KeyBackspace:
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
		m.snapToFilter()
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil

	case tea.KeyRunes:
		m.filter += string(msg.Runes)
		m.snapToFilter()
		m.onSelectionChange()
		m.refreshOutput()
		return m, nil
	}
	return m, nil
}

func (m Model) handleVerified(msg verifiedMsg) (tea.Model, tea.Cmd) {
	// ignore stale results for an exercise we've navigated away from
	if msg.name != m.current().Name {
		return m, nil
	}
	m.verifying = false
	m.status = msg.status
	m.result = msg.result
	m.hasResult = true

	// On success, record it and update the glyph/progress, but stay on this
	// exercise so the learner sees the result. They press `n` for the next one.
	if msg.status == exercises.StatusDone && !m.tracker.IsDone(msg.name) {
		m.tracker.MarkDone(msg.name, now())
		_ = m.tracker.Save()
		m.refreshHeaderCounts()
	}

	m.refreshOutput()
	return m, nil
}

// onSelectionChange resets per-exercise detail state when the cursor moves.
func (m *Model) onSelectionChange() {
	m.showHint = false
	m.hasResult = false
	m.notice = ""
	m.result = exercises.Result{}
	m.output.GotoTop()
}
