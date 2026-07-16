package exercises

import (
	"path/filepath"
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return tm
}

func TestLoadStateMissingFile(t *testing.T) {
	s, err := LoadState(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.DoneCount() != 0 || s.Completed == nil {
		t.Errorf("expected empty ready state, got %+v", s)
	}
}

func TestMarkDoneAndStreak(t *testing.T) {
	s, _ := LoadState(filepath.Join(t.TempDir(), "s.json"))

	if s.Streak() != 0 {
		t.Fatalf("empty streak should be 0, got %d", s.Streak())
	}

	day1 := mustTime(t, "2026-06-20T09:00:00Z")
	s.MarkDone("variables1", day1)
	if !s.IsDone("variables1") || s.Streak() != 1 {
		t.Fatalf("after first: done=%v streak=%d", s.IsDone("variables1"), s.Streak())
	}

	// same day, another completion -> streak unchanged
	s.MarkDone("variables2", day1.Add(time.Hour))
	if s.Streak() != 1 {
		t.Errorf("same-day streak should stay 1, got %d", s.Streak())
	}

	// next day -> streak increments
	s.MarkDone("functions1", mustTime(t, "2026-06-21T08:00:00Z"))
	if s.Streak() != 2 {
		t.Errorf("consecutive day streak should be 2, got %d", s.Streak())
	}

	// gap of a day then complete -> run ending at latest is just 1
	s.MarkDone("functions2", mustTime(t, "2026-06-23T08:00:00Z"))
	if s.Streak() != 1 {
		t.Errorf("after a gap streak should be 1, got %d", s.Streak())
	}

	// removing the latest completion drops the streak back to the earlier run
	s.Unmark("functions2")
	if s.Streak() != 2 {
		t.Errorf("after unmark streak should be 2 (20th+21st), got %d", s.Streak())
	}
}

func TestMarkDoneIdempotent(t *testing.T) {
	s, _ := LoadState(filepath.Join(t.TempDir(), "s.json"))
	first := mustTime(t, "2026-06-20T09:00:00Z")
	s.MarkDone("x", first)
	ts := s.Completed["x"]

	// re-marking later must not change the recorded timestamp or streak
	s.MarkDone("x", mustTime(t, "2026-06-25T09:00:00Z"))
	if s.Completed["x"] != ts {
		t.Errorf("timestamp changed on re-mark: %q -> %q", ts, s.Completed["x"])
	}
	if s.Streak() != 1 {
		t.Errorf("streak changed on re-mark: %d", s.Streak())
	}
}

func TestUnmark(t *testing.T) {
	s, _ := LoadState(filepath.Join(t.TempDir(), "s.json"))
	s.MarkDone("x", mustTime(t, "2026-06-20T09:00:00Z"))
	if !s.IsDone("x") {
		t.Fatal("precondition: x should be done")
	}
	s.Unmark("x")
	if s.IsDone("x") || s.DoneCount() != 0 {
		t.Errorf("after Unmark, x should not be done; count=%d", s.DoneCount())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	s, _ := LoadState(path)
	s.MarkDone("variables1", mustTime(t, "2026-06-20T09:00:00Z"))
	s.Current = "variables2"
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.IsDone("variables1") || loaded.Current != "variables2" || loaded.Streak() != 1 {
		t.Errorf("round trip lost data: %+v", loaded)
	}
}

func TestReconcileBackfills(t *testing.T) {
	s, _ := LoadState(filepath.Join(t.TempDir(), "s.json"))
	now := mustTime(t, "2026-06-22T10:00:00Z")
	s.Reconcile([]string{"a", "b", "c"}, now)
	if s.DoneCount() != 3 {
		t.Errorf("expected 3 backfilled, got %d", s.DoneCount())
	}
}
