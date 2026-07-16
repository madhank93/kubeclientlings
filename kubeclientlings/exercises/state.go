package exercises

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"
)

// StateFile is the default progress file name (kept beside info.toml).
const StateFile = ".clientlings-state.json"

const dayLayout = "2006-01-02"

// Tracker is persisted metadata about progress. The file markers ("I AM NOT
// DONE") remain the source of truth for done-ness; this only records when each
// exercise was first completed and the last position. The streak is derived
// from the completion timestamps, never stored, so it can't drift.
type Tracker struct {
	// Completed maps exercise name -> RFC3339 timestamp of first completion.
	Completed map[string]string `json:"completed"`
	Current   string            `json:"current"`

	path string
}

// LoadState reads the state file. A missing file yields an empty, ready State.
func LoadState(path string) (*Tracker, error) {
	s := &Tracker{Completed: map[string]string{}, path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return s, err
	}
	if s.Completed == nil {
		s.Completed = map[string]string{}
	}
	s.path = path
	return s, nil
}

// Save writes the state file (pretty-printed).
func (s *Tracker) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// IsDone reports whether the exercise has a recorded completion.
func (s *Tracker) IsDone(name string) bool {
	_, ok := s.Completed[name]
	return ok
}

// DoneCount returns how many exercises have been completed.
func (s *Tracker) DoneCount() int { return len(s.Completed) }

// Unmark clears an exercise's completion (used by reset). The streak, being a
// historical record of active days, is intentionally left unchanged.
func (s *Tracker) Unmark(name string) { delete(s.Completed, name) }

// MarkDone records a first-time completion at now.
// Re-marking an already-completed exercise is a no-op.
func (s *Tracker) MarkDone(name string, now time.Time) {
	if _, ok := s.Completed[name]; ok {
		return
	}
	s.Completed[name] = now.Format(time.RFC3339)
}

// Streak is the number of consecutive days, ending at the most recent
// completion, on which at least one exercise was completed. Derived from the
// completion timestamps, so resetting an exercise lowers it automatically.
func (s *Tracker) Streak() int {
	days := map[string]bool{}
	latest := ""
	for _, ts := range s.Completed {
		tm, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		d := tm.Format(dayLayout)
		days[d] = true
		if d > latest { // YYYY-MM-DD sorts chronologically
			latest = d
		}
	}
	if latest == "" {
		return 0
	}
	cur, _ := time.Parse(dayLayout, latest)
	count := 0
	for days[cur.Format(dayLayout)] {
		count++
		cur = cur.AddDate(0, 0, -1)
	}
	return count
}

// Reconcile backfills completions for exercises that are already Done on disk
// but have no recorded timestamp (e.g. finished before the state file existed).
func (s *Tracker) Reconcile(doneNames []string, now time.Time) {
	for _, name := range doneNames {
		s.MarkDone(name, now)
	}
}
