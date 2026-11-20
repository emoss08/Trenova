package simclient

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

const reconnectCeiling = 5 * time.Second

var ErrDeadline = errors.New("the deadline passed")

type entry struct {
	seq     uint64
	message journal.Message
}

type Watch struct {
	client   *Client
	mu       sync.Mutex
	byID     map[string]*entry
	byKey    map[string]map[string][]string
	seq      uint64
	last     uint64
	changed  chan struct{}
	failure  error
	stopped  chan struct{}
	stream   *http.Client
	cancel   context.CancelFunc
	finished sync.WaitGroup
}

func (c *Client) Watch(ctx context.Context) (*Watch, error) {
	w := &Watch{
		client:  c,
		byID:    map[string]*entry{},
		byKey:   map[string]map[string][]string{},
		changed: make(chan struct{}),
		stopped: make(chan struct{}),
		stream:  &http.Client{Transport: c.http.Transport},
	}
	streamCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	w.cancel = cancel
	connected := make(chan error, 1)
	w.finished.Go(func() { w.run(streamCtx, connected) })
	select {
	case err := <-connected:
		if err != nil {
			w.Close()
			return nil, err
		}
		return w, nil
	case <-ctx.Done():
		w.Close()
		return nil, ctx.Err()
	}
}

func (w *Watch) Close() {
	w.cancel()
	w.finished.Wait()
}

func (w *Watch) run(ctx context.Context, connected chan<- error) {
	defer close(w.stopped)
	first := true
	policy := backoff.NewExponentialBackOff()
	policy.MaxInterval = reconnectCeiling
	for {
		err := w.listen(ctx, func() {
			if first {
				first = false
				connected <- nil
			}
			policy.Reset()
		})
		if first {
			connected <- err
			return
		}
		if ctx.Err() != nil {
			return
		}
		w.mu.Lock()
		w.failure = err
		w.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(policy.NextBackOff()):
		}
	}
}

func (w *Watch) listen(ctx context.Context, onConnect func()) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		w.client.base+"/control/events", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/event-stream")
	w.mu.Lock()
	if w.last > 0 {
		request.Header.Set("Last-Event-ID", strconv.FormatUint(w.last, 10))
	}
	w.mu.Unlock()
	response, err := w.stream.Do(request)
	if err != nil {
		return fmt.Errorf("the simulator's event stream: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the simulator's event stream answered %d", response.StatusCode)
	}
	onConnect()
	return w.read(bufio.NewReaderSize(response.Body, 64<<10))
}

func (w *Watch) read(reader *bufio.Reader) error {
	var id uint64
	var data bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("the simulator closed its event stream")
			}
			return err
		}
		line = bytes.TrimRight(line, "\r\n")
		switch {
		case len(line) == 0:
			if data.Len() > 0 {
				w.apply(id, data.Bytes())
			}
			id = 0
			data.Reset()
		case bytes.HasPrefix(line, []byte("id:")):
			id, _ = strconv.ParseUint(strings.TrimSpace(string(line[3:])), 10, 64)
		case bytes.HasPrefix(line, []byte("data:")):
			data.Write(bytes.TrimPrefix(line[5:], []byte(" ")))
		}
	}
}

func (w *Watch) apply(id uint64, data []byte) {
	var event journal.Event
	if err := sonic.Unmarshal(data, &event); err != nil {
		return
	}
	w.mu.Lock()
	if id > w.last {
		w.last = id
	}
	if event.Message != nil {
		w.store(*event.Message)
	}
	close(w.changed)
	w.changed = make(chan struct{})
	w.mu.Unlock()
}

func (w *Watch) store(m journal.Message) {
	stored, ok := w.byID[m.ID]
	if !ok {
		w.seq++
		stored = &entry{seq: w.seq}
		w.byID[m.ID] = stored
	}
	previous := stored.message.Keys
	stored.message = m
	keys := w.byKey[m.Partner]
	if keys == nil {
		keys = map[string][]string{}
		w.byKey[m.Partner] = keys
	}
	for _, key := range m.Keys {
		if !slices.Contains(previous, key) {
			keys[key] = append(keys[key], m.ID)
		}
	}
}

func (w *Watch) Observe(messages []journal.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for index := range messages {
		if _, ok := w.byID[messages[index].ID]; !ok {
			w.store(messages[index])
		}
	}
}

type View struct {
	w *Watch
}

func (v View) About(partnerName, key string) []journal.Message {
	ids := v.w.byKey[partnerName][key]
	out := make([]journal.Message, 0, len(ids))
	for _, id := range ids {
		out = append(out, v.w.byID[id].message)
	}
	slices.SortStableFunc(out, func(a, b journal.Message) int {
		return int(v.w.byID[a.ID].seq) - int(v.w.byID[b.ID].seq)
	})
	return out
}

func (v View) Message(id string) (journal.Message, bool) {
	stored, ok := v.w.byID[id]
	if !ok {
		return journal.Message{}, false
	}
	return stored.message, true
}

func (v View) Partner(partnerName string, match func(*journal.Message) bool) []journal.Message {
	var out []entry
	for _, stored := range v.w.byID {
		if stored.message.Partner == partnerName && (match == nil || match(&stored.message)) {
			out = append(out, *stored)
		}
	}
	slices.SortFunc(out, func(a, b entry) int { return int(a.seq) - int(b.seq) })
	messages := make([]journal.Message, 0, len(out))
	for index := range out {
		messages = append(messages, out[index].message)
	}
	return messages
}

func (w *Watch) Read(fn func(View)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fn(View{w: w})
}

func (w *Watch) Until(ctx context.Context, deadline time.Time, done func(View) bool) error {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for {
		w.mu.Lock()
		finished := done(View{w: w})
		changed := w.changed
		failure := w.failure
		w.mu.Unlock()
		if finished {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopped:
			return fmt.Errorf("the simulator's event stream stopped: %w", failure)
		case <-timer.C:
			return ErrDeadline
		case <-changed:
		}
	}
}
