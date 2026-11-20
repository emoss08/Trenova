package journal

import (
	"sync"
	"time"
)

const (
	DefaultReplay     = 2048
	subscriberBacklog = 256
)

type Event struct {
	ID      uint64     `json:"id"`
	Type    string     `json:"type"`
	Partner string     `json:"partner,omitempty"`
	At      time.Time  `json:"at"`
	Message *Message   `json:"message,omitempty"`
	Run     *RunNotice `json:"run,omitempty"`
}

type RunNotice struct {
	RunID    string   `json:"runId"`
	Scenario string   `json:"scenario,omitempty"`
	Keys     []string `json:"keys,omitempty"`
}

type Subscription struct {
	events  chan Event
	partner string
	broker  *Broker
	once    sync.Once
}

func (s *Subscription) Events() <-chan Event {
	return s.events
}

func (s *Subscription) Close() {
	s.broker.drop(s)
}

type Broker struct {
	mu     sync.Mutex
	last   uint64
	recent []Event
	size   int
	subs   map[*Subscription]struct{}
	now    func() time.Time
}

func NewBroker(replay int, now func() time.Time) *Broker {
	if replay < 1 {
		replay = DefaultReplay
	}
	if now == nil {
		now = time.Now
	}
	return &Broker{size: replay, subs: map[*Subscription]struct{}{}, now: now}
}

func (b *Broker) Last() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.last
}

func (b *Broker) Resume(last uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if last > b.last {
		b.last = last
	}
}

func (b *Broker) Publish(kind, partner string, m *Message) {
	b.publish(Event{Type: kind, Partner: partner, Message: m})
}

func (b *Broker) PublishRun(kind, partner string, notice *RunNotice) {
	b.publish(Event{Type: kind, Partner: partner, Run: notice})
}

func (b *Broker) publish(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.last++
	event.ID, event.At = b.last, b.now().UTC()
	b.recent = append(b.recent, event)
	if len(b.recent) > b.size {
		b.recent = b.recent[len(b.recent)-b.size:]
	}
	for sub := range b.subs {
		if !sub.wants(event) {
			continue
		}
		select {
		case sub.events <- event:
		default:
			b.remove(sub)
		}
	}
}

func (b *Broker) Subscribe(after uint64, partner string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := &Subscription{partner: partner, broker: b}
	var replay []Event
	for _, event := range b.recent {
		if event.ID > after && sub.wants(event) {
			replay = append(replay, event)
		}
	}
	sub.events = make(chan Event, subscriberBacklog+len(replay))
	for _, event := range replay {
		sub.events <- event
	}
	b.subs[sub] = struct{}{}
	return sub
}

func (b *Broker) drop(sub *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.remove(sub)
}

func (b *Broker) remove(sub *Subscription) {
	if _, ok := b.subs[sub]; !ok {
		return
	}
	delete(b.subs, sub)
	sub.once.Do(func() { close(sub.events) })
}

func (s *Subscription) wants(event Event) bool {
	return s.partner == "" || event.Partner == "" || event.Partner == s.partner
}
