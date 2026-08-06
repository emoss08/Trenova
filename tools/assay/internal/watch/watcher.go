package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/mod/modfile"
)

const defaultDebounce = 250 * time.Millisecond

// maxDebounceStretch bounds how long a continuous writer can postpone a cycle.
// A codegen loop that saves every 100ms would otherwise reset the quiet-period
// timer forever and the session would never run anything.
const maxDebounceStretch = 4

// Watcher turns raw filesystem events into debounced batches of relevant paths.
//
// Editors are noisy — atomic-save temp files, backup files, a write event per
// chunk — so events are filtered to files that can change what runs (.go and
// module files) and collected until the tree has been quiet for the debounce
// window, or until a continuous writer has stretched the wait to its cap.
//
// A batch send never blocks the event pump: if the consumer is mid-cycle and
// the channel is full, the paths stay in pending and ride out with the next
// flush. Overflow is the one event that cannot be papered over — the kernel
// dropped an unknown number of changes — so it surfaces as a resync batch (nil
// paths), which the planner treats as "run the whole dirty plan" and the loop
// treats as "drop the memoised graph".
type Watcher struct {
	fs       *fsnotify.Watcher
	root     string
	debounce time.Duration
	batches  chan Batch
}

// Batch is one debounced set of saves. Resync means the paths are unreliable —
// events were dropped — and the whole dirty tree must be reconsidered from
// scratch.
type Batch struct {
	Paths  []string
	Resync bool
}

func NewWatcher(root string, debounce time.Duration) (*Watcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("start filesystem watcher: %w", err)
	}
	if debounce <= 0 {
		debounce = defaultDebounce
	}

	w := &Watcher{fs: inner, root: root, debounce: debounce, batches: make(chan Batch, 4)}
	for _, tree := range watchRoots(root) {
		if _, err := w.addTree(tree); err != nil {
			inner.Close()

			if errors.Is(err, syscall.ENOSPC) {
				return nil, fmt.Errorf(
					"watch %s: %w (inotify watch limit; raise fs.inotify.max_user_watches)", tree, err)
			}

			return nil, err
		}
	}
	if err := inner.Add(root); err != nil {
		inner.Close()

		return nil, fmt.Errorf("watch %s: %w", root, err)
	}

	return w, nil
}

// watchRoots limits the watch to the trees that can hold Go packages. In a
// polyglot monorepo the git root drags in every frontend and asset directory —
// on this workspace 435 of 1,198 watched directories were a TypeScript tree
// that can never produce a relevant event but happily produces thousands of
// irrelevant ones during a frontend build, starving the reader. When a go.work
// names module roots, those are the watch set (plus the root itself, shallow,
// for go.work edits); without one, the whole root is watched as before.
func watchRoots(root string) []string {
	work, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return []string{root}
	}

	parsed, err := modfile.ParseWork("go.work", work, nil)
	if err != nil {
		return []string{root}
	}

	var roots []string
	for _, use := range parsed.Use {
		dir := use.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, filepath.FromSlash(dir))
		}
		roots = append(roots, filepath.Clean(dir))
	}
	if len(roots) == 0 {
		return []string{root}
	}

	return roots
}

func (w *Watcher) Batches() <-chan Batch { return w.batches }

func (w *Watcher) Close() error { return w.fs.Close() }

// Run pumps events until the context ends.
func (w *Watcher) Run(ctx context.Context) {
	pending := make(map[string]struct{})
	resync := false
	var stretchedSince time.Time

	// Reset on an unbuffered-channel timer is safe without draining under the
	// Go 1.23+ semantics this module's go directive guarantees; downgrading the
	// directive would quietly reintroduce stale fires here.
	var timer *time.Timer
	var fire <-chan time.Time

	arm := func() {
		if timer == nil {
			timer = time.NewTimer(w.debounce)
			fire = timer.C
		} else {
			timer.Reset(w.debounce)
		}
		if stretchedSince.IsZero() {
			stretchedSince = time.Now()
		}
	}

	flush := func() {
		if len(pending) == 0 && !resync {
			return
		}

		batch := Batch{Resync: resync}
		for path := range pending {
			batch.Paths = append(batch.Paths, path)
		}

		select {
		case w.batches <- batch:
			pending = make(map[string]struct{})
			resync = false
			stretchedSince = time.Time{}
		default:
			// The consumer is mid-cycle and the queue is full. Keep everything in
			// pending — the paths ride out with a later flush — rather than
			// blocking this loop and letting the kernel queue overflow behind us.
			arm()
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

			// Renames deliver as Create on Linux, so an atomic-save editor's whole
			// signal for a modified file is one Create event. Files are recorded as
			// changes; directories are walked for new watches and any files already
			// inside them — written between the mkdir and our Add — are recorded too.
			if event.Op.Has(fsnotify.Create) {
				if info, statErr := os.Lstat(event.Name); statErr == nil && info.IsDir() {
					seeded, _ := w.addTree(event.Name)
					for _, path := range seeded {
						pending[path] = struct{}{}
					}
					if len(seeded) > 0 {
						arm()
					}

					continue
				}
			}
			if !relevantFile(event.Name) {
				continue
			}
			pending[filepath.Clean(event.Name)] = struct{}{}

			if !stretchedSince.IsZero() &&
				time.Since(stretchedSince) > time.Duration(maxDebounceStretch)*w.debounce {
				flush()

				continue
			}
			arm()
		case <-fire:
			flush()
		case err, open := <-w.fs.Errors:
			if !open {
				continue
			}
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				resync = true
				arm()
			}
		}
	}
}

// addTree watches a directory tree and returns the relevant files already in
// it, so a directory that appears mid-session (git checkout, code generation)
// contributes the files written before our watch landed.
func (w *Watcher) addTree(root string) ([]string, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	var seeded []string
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, inner error) error {
		if inner != nil {
			return nil
		}
		if !entry.IsDir() {
			if path != root && relevantFile(path) {
				seeded = append(seeded, filepath.Clean(path))
			}

			return nil
		}
		name := entry.Name()
		if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
			return fs.SkipDir
		}

		return w.fs.Add(path)
	})

	return seeded, walkErr
}

func relevantFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") {
		return false
	}
	if strings.HasSuffix(base, ".go") {
		return true
	}
	if strings.Contains(path, string(filepath.Separator)+"testdata"+string(filepath.Separator)) {
		return true
	}
	switch base {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}
