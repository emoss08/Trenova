// Package tailbuf bounds a stream to its most recent bytes. Harness processes
// hold one stderr buffer for a whole package's tests; unbounded, a chatty suite
// multiplies the resident cost by the number of concurrent workers, and only
// the tail ever reaches an error message.
package tailbuf

// Buffer keeps only the last limit bytes written to it.
type Buffer struct {
	limit int
	data  []byte
}

func New(limit int) *Buffer {
	return &Buffer{limit: limit}
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if len(b.data) > b.limit {
		b.data = b.data[len(b.data)-b.limit:]
	}

	return len(p), nil
}

func (b *Buffer) String() string {
	return string(b.data)
}
