package audit

import (
	"sync"

	"github.com/trenova-app/transport/internal/core/domain/audit"
)

type Buffer struct {
	// Entries is a list of audit entries.
	entries []*audit.Entry

	// mu is a mutex to synchronize access to the buffer.
	mu sync.Mutex

	// limit is the maximum number of entries in the buffer.
	limit int
}

func NewBuffer(limit int) *Buffer {
	return &Buffer{
		entries: make([]*audit.Entry, 0, limit),
		limit:   limit,
	}
}

// Add adds an audit entry to the buffer.
func (b *Buffer) Add(entry *audit.Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries = append(b.entries, entry)
}

// Flush returns the current entries in the buffer and resets the buffer.
func (b *Buffer) FlushAndReset() []*audit.Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		return nil
	}

	entries := make([]*audit.Entry, len(b.entries))
	copy(entries, b.entries)

	// Reset the buffer
	b.entries = make([]*audit.Entry, 0, b.limit)

	return entries
}

// IsFull returns true if the buffer is full.
func (b *Buffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries) >= b.limit
}

// Size returns the number of entries in the buffer.
func (b *Buffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}
