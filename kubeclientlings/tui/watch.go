package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// debounce is how long the watcher waits for writes to settle before firing.
// Editors rarely save once: many write, rename and chmod the same file in a
// burst, and a naive watcher re-runs the exercise for each event.
const debounce = 400 * time.Millisecond

// fileChangedMsg is emitted when a watched exercise file settles after a save.
type fileChangedMsg struct{}

// watchRelevant reports whether a changed path should trigger a re-verify.
// Editor swap files and dotfiles churn constantly and never affect the
// outcome of a run.
func watchRelevant(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}
	switch filepath.Ext(path) {
	case ".go", ".mod", ".sum":
		return true
	}
	return false
}

// startWatcher watches dir recursively and pushes a debounced fileChangedMsg
// into ch. The returned watcher is the caller's to Close.
//
// Directories created after startup (a new exercise pulled down while the TUI
// is open) are picked up as they appear: fsnotify watches are per-directory
// and do not recurse, so each new dir has to be added explicitly.
func startWatcher(dir string, ch chan tea.Msg) (*fsnotify.Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	addTree(w, dir)

	go func() {
		var timer *time.Timer
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Create) {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						addTree(w, ev.Name)
					}
				}
				if !watchRelevant(ev.Name) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, func() { ch <- fileChangedMsg{} })

			// fsnotify's Errors channel must be drained. Left unread it fills
			// and the library stops delivering events, which silently killed
			// watch mode: the TUI stayed up but no longer reacted to saves.
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return w, nil
}

// addTree adds dir and every directory beneath it to the watcher.
func addTree(w *fsnotify.Watcher, dir string) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr // an unreadable subtree is not fatal
		}
		if strings.HasPrefix(d.Name(), ".") && path != dir {
			return filepath.SkipDir
		}
		_ = w.Add(path)
		return nil
	})
}

// waitForChange blocks on ch and turns the next event into a tea.Msg.
func waitForChange(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}
