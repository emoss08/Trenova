package schedule

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"
)

type Clock interface {
	Now() time.Time
	Until(at time.Time) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Until(at time.Time) <-chan time.Time { return time.After(time.Until(at)) }

func RealClock() Clock { return realClock{} }

type job struct {
	at  time.Time
	seq uint64
	run func(context.Context)
}

type jobs []*job

func (j jobs) Len() int { return len(j) }

func (j jobs) Less(a, b int) bool {
	if j[a].at.Equal(j[b].at) {
		return j[a].seq < j[b].seq
	}
	return j[a].at.Before(j[b].at)
}

func (j jobs) Swap(a, b int) { j[a], j[b] = j[b], j[a] }

func (j *jobs) Push(x any) { *j = append(*j, x.(*job)) }

func (j *jobs) Pop() any {
	old := *j
	last := old[len(old)-1]
	old[len(old)-1] = nil
	*j = old[:len(old)-1]
	return last
}

type Scheduler struct {
	clock   Clock
	workers int
	mu      sync.Mutex
	queue   jobs
	seq     uint64
	wake    chan struct{}
}

func New(clock Clock, workers int) *Scheduler {
	if clock == nil {
		clock = RealClock()
	}
	return &Scheduler{clock: clock, workers: max(workers, 1), wake: make(chan struct{}, 1)}
}

func (s *Scheduler) Now() time.Time {
	return s.clock.Now()
}

func (s *Scheduler) At(at time.Time, run func(context.Context)) {
	s.mu.Lock()
	s.seq++
	heap.Push(&s.queue, &job{at: at, seq: s.seq, run: run})
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Scheduler) After(d time.Duration, run func(context.Context)) {
	s.At(s.clock.Now().Add(d), run)
}

func (s *Scheduler) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

func (s *Scheduler) Clear() {
	s.mu.Lock()
	s.queue = nil
	s.mu.Unlock()
}

func (s *Scheduler) Run(ctx context.Context) {
	workers := pool.New().WithContext(ctx).WithMaxGoroutines(s.workers)
	defer func() { _ = workers.Wait() }()
	for {
		due, next, waiting := s.due()
		for _, j := range due {
			run := j.run
			workers.Go(func(ctx context.Context) error {
				run(ctx)
				return nil
			})
		}
		var timer <-chan time.Time
		if waiting {
			timer = s.clock.Until(next)
		}
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-timer:
		}
	}
}

func (s *Scheduler) due() ([]*job, time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	var ready []*job
	for len(s.queue) > 0 && !s.queue[0].at.After(now) {
		ready = append(ready, heap.Pop(&s.queue).(*job))
	}
	if len(s.queue) == 0 {
		return ready, time.Time{}, false
	}
	return ready, s.queue[0].at, true
}
