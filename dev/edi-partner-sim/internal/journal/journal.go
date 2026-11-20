package journal

import (
	"cmp"
	"fmt"
	"slices"
	"sync"
	"time"
)

const DefaultCapacity = 1000

type Filter struct {
	Direction Direction
	Set       string
	Key       string
	Limit     int
}

type Conversation struct {
	Key      string    `json:"key"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
	Sets     []string  `json:"sets"`
	Messages []Summary `json:"messages"`
}

type ring struct {
	items []*Message
	start int
	count int
}

func (r *ring) push(m *Message) *Message {
	if r.count < len(r.items) {
		r.items[(r.start+r.count)%len(r.items)] = m
		r.count++
		return nil
	}
	evicted := r.items[r.start]
	r.items[r.start] = m
	r.start = (r.start + 1) % len(r.items)
	return evicted
}

func (r *ring) each(fn func(*Message) bool) {
	for index := range r.count {
		if !fn(r.items[(r.start+index)%len(r.items)]) {
			return
		}
	}
}

func (r *ring) newestFirst(fn func(*Message) bool) {
	for index := r.count - 1; index >= 0; index-- {
		if !fn(r.items[(r.start+index)%len(r.items)]) {
			return
		}
	}
}

type Journal struct {
	mu       sync.RWMutex
	capacity int
	seq      uint64
	docSeq   uint64
	version  uint64
	rings    map[string]*ring
	byID     map[string]*Message
	events   *Broker
	now      func() time.Time
}

func New(capacity int, events *Broker, now func() time.Time) *Journal {
	if capacity < 1 {
		capacity = DefaultCapacity
	}
	if now == nil {
		now = time.Now
	}
	return &Journal{
		capacity: capacity,
		rings:    map[string]*ring{},
		byID:     map[string]*Message{},
		events:   events,
		now:      now,
	}
}

func (j *Journal) Events() *Broker {
	return j.events
}

func (j *Journal) Version() uint64 {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.version
}

func (j *Journal) Add(event string, m *Message) Message {
	j.mu.Lock()
	j.seq++
	m.ID = fmt.Sprintf("%s-%06d", m.Partner, j.seq)
	if m.At.IsZero() {
		m.At = j.now().UTC()
	}
	for index := range m.Documents {
		j.docSeq++
		m.Documents[index].Number = j.docSeq
	}
	j.insert(m)
	j.version++
	snapshot := m.Clone()
	j.mu.Unlock()
	j.events.Publish(event, m.Partner, &snapshot)
	return snapshot
}

func (j *Journal) insert(m *Message) {
	r, ok := j.rings[m.Partner]
	if !ok {
		r = &ring{items: make([]*Message, j.capacity)}
		j.rings[m.Partner] = r
	}
	if evicted := r.push(m); evicted != nil {
		delete(j.byID, evicted.ID)
	}
	j.byID[m.ID] = m
}

func (j *Journal) Update(event, id string, fn func(*Message)) (Message, bool) {
	j.mu.Lock()
	m, ok := j.byID[id]
	if !ok {
		j.mu.Unlock()
		return Message{}, false
	}
	fn(m)
	j.version++
	snapshot := m.Clone()
	j.mu.Unlock()
	if event != "" {
		j.events.Publish(event, m.Partner, &snapshot)
	}
	return snapshot, true
}

func (j *Journal) Get(id string) (Message, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	m, ok := j.byID[id]
	if !ok {
		return Message{}, false
	}
	return m.Clone(), true
}

func (j *Journal) Find(partner string, match func(*Message) bool) (Message, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	r, ok := j.rings[partner]
	if !ok {
		return Message{}, false
	}
	var found *Message
	r.newestFirst(func(m *Message) bool {
		if match(m) {
			found = m
			return false
		}
		return true
	})
	if found == nil {
		return Message{}, false
	}
	return found.Clone(), true
}

func (j *Journal) Messages(partner string, filter Filter) []Message {
	j.mu.RLock()
	defer j.mu.RUnlock()
	r, ok := j.rings[partner]
	if !ok {
		return []Message{}
	}
	out := []Message{}
	r.newestFirst(func(m *Message) bool {
		if filter.matches(m) {
			out = append(out, m.Clone())
		}
		return filter.Limit <= 0 || len(out) < filter.Limit
	})
	slices.Reverse(out)
	return out
}

func (f Filter) matches(m *Message) bool {
	if f.Direction != "" && m.Direction != f.Direction {
		return false
	}
	if f.Set != "" && !slices.Contains(m.Sets, f.Set) {
		return false
	}
	return f.Key == "" || slices.Contains(m.Keys, f.Key)
}

func (j *Journal) Conversations(partner string) []Conversation {
	j.mu.RLock()
	defer j.mu.RUnlock()
	r, ok := j.rings[partner]
	if !ok {
		return []Conversation{}
	}
	index := map[string]*Conversation{}
	r.each(func(m *Message) bool {
		for _, key := range m.Keys {
			at := m.WireAt()
			c, seen := index[key]
			if !seen {
				c = &Conversation{Key: key, First: at, Last: at, Sets: []string{}}
				index[key] = c
			}
			if at.Before(c.First) {
				c.First = at
			}
			if at.After(c.Last) {
				c.Last = at
			}
			for _, set := range m.Sets {
				if !slices.Contains(c.Sets, set) {
					c.Sets = append(c.Sets, set)
				}
			}
			c.Messages = append(c.Messages, m.Summary())
		}
		return true
	})
	out := make([]Conversation, 0, len(index))
	for _, c := range index {
		out = append(out, *c)
	}
	slices.SortFunc(out, func(a, b Conversation) int {
		return cmp.Or(b.Last.Compare(a.Last), cmp.Compare(a.Key, b.Key))
	})
	return out
}

func (j *Journal) Pending() []Message {
	j.mu.RLock()
	defer j.mu.RUnlock()
	var out []Message
	for _, m := range j.byID {
		if m.State == StateScheduled {
			out = append(out, m.Clone())
		}
	}
	slices.SortFunc(out, func(a, b Message) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

func (j *Journal) Reset(partner string) {
	j.mu.Lock()
	for name, r := range j.rings {
		if partner != "" && name != partner {
			continue
		}
		r.each(func(m *Message) bool {
			delete(j.byID, m.ID)
			return true
		})
		delete(j.rings, name)
	}
	j.version++
	j.mu.Unlock()
	j.events.Publish("reset", partner, nil)
}

type Snapshot struct {
	Seq      uint64    `json:"seq"`
	DocSeq   uint64    `json:"docSeq"`
	EventSeq uint64    `json:"eventSeq"`
	Messages []Message `json:"messages"`
}

func (j *Journal) Snapshot() Snapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	snapshot := Snapshot{
		Seq:      j.seq,
		DocSeq:   j.docSeq,
		EventSeq: j.events.Last(),
		Messages: []Message{},
	}
	for _, name := range sortedKeys(j.rings) {
		j.rings[name].each(func(m *Message) bool {
			snapshot.Messages = append(snapshot.Messages, m.Clone())
			return true
		})
	}
	return snapshot
}

func (j *Journal) Restore(snapshot Snapshot) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.seq = snapshot.Seq
	j.docSeq = snapshot.DocSeq
	j.rings = map[string]*ring{}
	j.byID = map[string]*Message{}
	for index := range snapshot.Messages {
		m := snapshot.Messages[index].Clone()
		for _, d := range m.Documents {
			j.docSeq = max(j.docSeq, d.Number)
		}
		j.insert(&m)
	}
	j.events.Resume(snapshot.EventSeq)
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
