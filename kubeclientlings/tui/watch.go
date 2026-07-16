package tui

import (
	"io/fs"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// fileChangedMsg is emitted when an exercise file is written.
type fileChangedMsg struct{ path string }

// startWatcher walks exercises/ and forwards write/rename events onto ch.
// It runs for the life of the program.
func startWatcher(ch chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	root, _ := os.Getwd()
	dir := filepath.Join(root, "exercises")
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = watcher.Add(p)
		}
		return nil
	})

	for event := range watcher.Events {
		if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
			ch <- event.Name
		}
	}
}

// waitForChange blocks on ch and turns the next event into a tea.Msg.
func waitForChange(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return fileChangedMsg{path: <-ch}
	}
}
