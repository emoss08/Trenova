package captureservice

import (
	"sync"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// receivedAnnounceEvery is how often a batch taking pages is announced.
	// A scanner feeds a page every second or two, and every announcement
	// makes each open queue and the open batch refetch, so a 500-page scan
	// announced per page is 500 refetches on every screen watching.
	receivedAnnounceEvery = 3 * time.Second
	// receivedWindowsKept is how many batches' windows are kept before the
	// ones long quiet are dropped.
	receivedWindowsKept = 256
)

// receivedAnnouncer announces pages arriving on a batch at most once per
// window, and once more at the window's end when pages arrived inside it, so
// the last page is always shown. Each API instance keeps its own windows.
type receivedAnnouncer struct {
	mu      sync.Mutex
	every   time.Duration
	windows map[pulid.ID]*receivedWindow
}

type receivedWindow struct {
	last    time.Time
	pending bool
}

func newReceivedAnnouncer(every time.Duration) *receivedAnnouncer {
	return &receivedAnnouncer{
		every:   every,
		windows: make(map[pulid.ID]*receivedWindow),
	}
}

// announce calls publish now, or at the end of the batch's window when it
// was announced within it. A nil announcer publishes every time.
func (a *receivedAnnouncer) announce(batchID pulid.ID, publish func()) {
	if a == nil {
		publish()
		return
	}

	a.mu.Lock()
	now := time.Now()
	window, seen := a.windows[batchID]
	if !seen || now.Sub(window.last) >= a.every {
		if !seen {
			a.pruneLocked(now)
			window = &receivedWindow{}
			a.windows[batchID] = window
		}
		window.last = now
		a.mu.Unlock()
		publish()
		return
	}
	if window.pending {
		a.mu.Unlock()
		return
	}
	window.pending = true
	wait := a.every - now.Sub(window.last)
	a.mu.Unlock()

	time.AfterFunc(wait, func() {
		a.mu.Lock()
		window.pending = false
		window.last = time.Now()
		a.mu.Unlock()
		publish()
	})
}

// forget drops a batch's window once it stops taking pages.
func (a *receivedAnnouncer) forget(batchID pulid.ID) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if window, ok := a.windows[batchID]; ok && !window.pending {
		delete(a.windows, batchID)
	}
}

func (a *receivedAnnouncer) pruneLocked(now time.Time) {
	if len(a.windows) < receivedWindowsKept {
		return
	}
	for id, window := range a.windows {
		if !window.pending && now.Sub(window.last) >= a.every {
			delete(a.windows, id)
		}
	}
}
