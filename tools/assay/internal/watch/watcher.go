package watch

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 250 * time.Millisecond

// Watcher turns raw filesystem events into debounced batches of relevant paths.
//
// Editors are noisy — atomic-save temp files, backup files, a write event per
// chunk — so events are filtered to files that can change what runs (.go and
// module files) and collected until the tree has been quiet for the debounce
// window. One save, one batch, one cycle.
type Watcher struct {
	fs       *fsnotify.Watcher
	root     string
	debounce time.Duration
	batches  chan []string
}

func NewWatcher(root string, debounce time.Duration) (*Watcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("start filesystem watcher: %w", err)
	}
	if debounce <= 0 {
		debounce = defaultDebounce
	}

	w := &Watcher{fs: inner, root: root, debounce: debounce, batches: make(chan []string, 4)}
	if err := w.addTree(root); err != nil {
		inner.Close()

		return nil, err
	}

	return w, nil
}

func (w *Watcher) Batches() <-chan []string { return w.batches }

func (w *Watcher) Close() error { return w.fs.Close() }

// Run pumps events until the context ends. It never drops a relevant change: if
// a batch is pending while the consumer is still busy, the next batch absorbs
// the accumulated paths.
func (w *Watcher) Run(ctx context.Context) {
	pending := make(map[string]struct{})
	var timer *time.Timer
	var fire <-chan time.Time

	flush := func() {
		if len(pending) == 0 {
			return
		}
		batch := make([]string, 0, len(pending))
		for path := range pending {
			batch = append(batch, path)
		}
		pending = make(map[string]struct{})

		select {
		case w.batches <- batch:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			close(w.batches)

			return
		case event, open := <-w.fs.Events:
			if !open {
				flush()
				close(w.batches)

				return
			}

			if event.Op.Has(fsnotify.Create) {
				// A new directory needs its own watch; a new file is a change.
				if err := w.addTree(event.Name); err == nil {
					continue
				}
			}
			if !relevantFile(event.Name) {
				continue
			}
			pending[filepath.Clean(event.Name)] = struct{}{}

			if timer == nil {
				timer = time.NewTimer(w.debounce)
				fire = timer.C
			} else {
				timer.Reset(w.debounce)
			}
		case <-fire:
			flush()
		case <-w.fs.Errors:
		}
	}
}

// addTree watches a directory and everything under it. Files pass through
// untouched: fsnotify watches directories, and a path that is not a directory
// is simply an event source we already cover.
func (w *Watcher) addTree(root string) error {
	info, err := filepath.Glob(root)
	if err != nil || len(info) == 0 {
		return nil
	}

	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
			return fs.SkipDir
		}

		return w.fs.Add(path)
	})
}

func relevantFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") {
		return false
	}
	if strings.HasSuffix(base, ".go") {
		return true
	}
	switch base {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}
