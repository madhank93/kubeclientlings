package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madhank93/kubeclientlings/kubeclientlings/exercises"
)

// cFooter is a readable foreground for the keybind bar on light/dark.
var cFooter = lipgloss.AdaptiveColor{Light: "#24292f", Dark: "#c9d1d9"}

// phase is which screen the TUI is showing.
type phase int

const (
	phaseWelcome phase = iota
	phaseMain
)

// item is one row in the left pane: either a topic header or an exercise.
type item struct {
	isHeader    bool
	topic       string
	done, total int // header only
	ex          exercises.Exercise
}

// Model is the Bubble Tea model for the clientlings TUI.
type Model struct {
	infoFile string
	tracker  *exercises.Tracker
	watchCh  chan string

	phase phase

	items  []item
	cursor int // index into items; always points at an exercise row

	// detail of the currently selected exercise
	status    exercises.Status
	result    exercises.Result
	verifying bool
	hasResult bool
	showHint  bool

	keys     keyMap
	help     help.Model
	progress progress.Model
	spinner  spinner.Model
	output   viewport.Model

	filtering bool   // typing a search query in the list pane
	filter    string // current search query

	total  int
	width  int
	height int

	notice string // transient message (e.g. after reset)
}

// matchesFilter reports whether an exercise matches the current search query
// (case-insensitive substring on its name or topic). An empty query matches all.
func (m Model) matchesFilter(ex exercises.Exercise) bool {
	if m.filter == "" {
		return true
	}
	q := strings.ToLower(m.filter)
	return strings.Contains(strings.ToLower(ex.Name), q) ||
		strings.Contains(strings.ToLower(topicOf(ex.Path)), q)
}

// snapToFilter keeps the cursor on a row that matches the query; if the current
// row no longer matches, it jumps to the first matching exercise.
func (m *Model) snapToFilter() {
	if m.cursor >= 0 && m.cursor < len(m.items) &&
		!m.items[m.cursor].isHeader && m.matchesFilter(m.items[m.cursor].ex) {
		return
	}
	for i, it := range m.items {
		if it.isHeader {
			continue
		}
		if m.matchesFilter(it.ex) {
			m.cursor = i
			return
		}
	}
}

// New builds the model, loads exercises and saved progress, and starts the
// file watcher goroutine.
func New(infoFile string) (Model, error) {
	exs, err := exercises.List(infoFile)
	if err != nil {
		return Model{}, err
	}
	tracker, err := exercises.LoadState(exercises.StateFile)
	if err != nil {
		return Model{}, err
	}

	ch := make(chan string)
	go startWatcher(ch)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(cTeal).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(cFooter)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(cDim)

	m := Model{
		infoFile: infoFile,
		tracker:  tracker,
		watchCh:  ch,
		keys:     defaultKeys(),
		help:     h,
		progress: progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		spinner:  sp,
		output:   viewport.New(0, 0),
		total:    len(exs),
	}
	m.items = buildItems(exs, tracker)
	m.cursor = m.firstSelectable()
	m.moveToFirstPending()
	return m, nil
}

// Init starts the watcher listener, the spinner, and verifies the current item.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForChange(m.watchCh),
		m.spinner.Tick,
		verifyCmd(m.current()),
	)
}

// buildItems flattens exercises into topic headers + exercise rows.
func buildItems(exs []exercises.Exercise, tracker *exercises.Tracker) []item {
	var items []item
	lastTopic := ""
	var headerIdx int
	for _, ex := range exs {
		topic := topicOf(ex.Path)
		if topic != lastTopic {
			items = append(items, item{isHeader: true, topic: topic})
			headerIdx = len(items) - 1
			lastTopic = topic
		}
		items = append(items, item{ex: ex})
		items[headerIdx].total++
		if tracker.IsDone(ex.Name) {
			items[headerIdx].done++
		}
	}
	return items
}

// current returns the exercise under the cursor.
func (m Model) current() exercises.Exercise {
	if m.cursor >= 0 && m.cursor < len(m.items) && !m.items[m.cursor].isHeader {
		return m.items[m.cursor].ex
	}
	return exercises.Exercise{}
}

func (m Model) firstSelectable() int {
	for i, it := range m.items {
		if !it.isHeader {
			return i
		}
	}
	return 0
}

// moveToFirstPending positions the cursor on the first not-yet-done exercise.
func (m *Model) moveToFirstPending() {
	for i, it := range m.items {
		if it.isHeader {
			continue
		}
		if !m.tracker.IsDone(it.ex.Name) {
			m.cursor = i
			return
		}
	}
}

// moveCursor steps to the next/previous exercise row, skipping headers (and,
// while filtering, any rows that don't match the query).
func (m *Model) moveCursor(delta int) {
	i := m.cursor
	for {
		i += delta
		if i < 0 || i >= len(m.items) {
			return // out of range; keep current
		}
		if m.items[i].isHeader {
			continue
		}
		if m.filtering && !m.matchesFilter(m.items[i].ex) {
			continue
		}
		m.cursor = i
		return
	}
}

// advance moves to the next not-done exercise after the cursor.
func (m *Model) advance() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if m.items[i].isHeader {
			continue
		}
		if !m.tracker.IsDone(m.items[i].ex.Name) {
			m.cursor = i
			return
		}
	}
}

// refreshHeaderCounts recomputes per-topic done counts from the tracker.
func (m *Model) refreshHeaderCounts() {
	var hdr int
	for i := range m.items {
		if m.items[i].isHeader {
			m.items[i].done = 0
			hdr = i
			continue
		}
		if m.tracker.IsDone(m.items[i].ex.Name) {
			m.items[hdr].done++
		}
	}
}

// verifiedMsg carries the result of verifying an exercise.
type verifiedMsg struct {
	name   string
	status exercises.Status
	result exercises.Result
}

// verifyCmd runs the gated verification off the UI thread.
func verifyCmd(e exercises.Exercise) tea.Cmd {
	if e.Name == "" {
		return nil
	}
	return func() tea.Msg {
		st, res := e.Verify()
		return verifiedMsg{name: e.Name, status: st, result: res}
	}
}

func now() time.Time { return time.Now() }
