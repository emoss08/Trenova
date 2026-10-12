package captureservice

import (
	"context"
	"sync"
	"time"
)

const (
	// arrivalReaderCount is how many pages each API instance reads at once as
	// they arrive. Reading renders the page, so it is kept small next to the
	// requests the instance is serving.
	arrivalReaderCount = 2
	// arrivalQueueLength is how many arrived pages may wait to be read. A page
	// that finds the queue full is read when its batch is sealed instead.
	arrivalQueueLength = 128
	// arrivalReadTimeout bounds one page's read.
	arrivalReadTimeout = 2 * time.Minute
)

// arrivalReaders read pages in the background as they are uploaded. They
// start with the first page and run for the life of the process.
type arrivalReaders struct {
	once    sync.Once
	readers int
	jobs    chan func(context.Context)
}

func newArrivalReaders(readers, queue int) *arrivalReaders {
	return &arrivalReaders{readers: readers, jobs: make(chan func(context.Context), queue)}
}

// submit queues a read, and reports whether it was queued. A nil set of
// readers queues nothing.
func (a *arrivalReaders) submit(read func(context.Context)) bool {
	if a == nil {
		return false
	}
	a.once.Do(a.start)
	select {
	case a.jobs <- read:
		return true
	default:
		return false
	}
}

func (a *arrivalReaders) start() {
	for range a.readers {
		go func() {
			for read := range a.jobs {
				ctx, cancel := context.WithTimeout(context.Background(), arrivalReadTimeout)
				read(ctx)
				cancel()
			}
		}()
	}
}
