package journal

import (
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
}

func TestTheRingKeepsTheNewestPerPartner(t *testing.T) {
	t.Parallel()

	j := New(2, NewBroker(10, fixedNow), fixedNow)
	first := j.Add(
		"message.received",
		&Message{Partner: "a", Sets: []string{"990"}, Keys: []string{"K1"}},
	)
	j.Add("message.received", &Message{Partner: "a", Sets: []string{"214"}, Keys: []string{"K1"}})
	j.Add("message.received", &Message{Partner: "a", Sets: []string{"210"}, Keys: []string{"K2"}})
	j.Add("message.received", &Message{Partner: "b", Sets: []string{"204"}})

	if _, ok := j.Get(first.ID); ok {
		t.Fatal("the oldest message was not evicted")
	}
	got := j.Messages("a", Filter{})
	if len(got) != 2 || got[0].Sets[0] != "214" || got[1].Sets[0] != "210" {
		t.Fatalf("partner a holds %+v", got)
	}
	if len(j.Messages("b", Filter{})) != 1 {
		t.Fatal("partner b lost its message to partner a's ring")
	}
	if got := j.Messages("a", Filter{Set: "210"}); len(got) != 1 {
		t.Fatalf("filter by set: %+v", got)
	}
	conversations := j.Conversations("a")
	if len(conversations) != 2 {
		t.Fatalf("conversations %+v", conversations)
	}
}

func TestUpdatesAreCopiedOutAndPublished(t *testing.T) {
	t.Parallel()

	broker := NewBroker(10, fixedNow)
	j := New(10, broker, fixedNow)
	sub := broker.Subscribe(0, "a")
	defer sub.Close()
	added := j.Add("message.received", &Message{Partner: "a"})
	j.Add("message.received", &Message{Partner: "b"})
	updated, ok := j.Update("ack.scheduled", added.ID, func(m *Message) {
		m.Ack = &Ack{Status: AckScheduled}
	})
	if !ok || updated.Ack.Status != AckScheduled {
		t.Fatalf("update: %+v", updated)
	}
	updated.Ack.Status = AckSent
	if again, _ := j.Get(added.ID); again.Ack.Status != AckScheduled {
		t.Fatal("a copy handed out shares its ack with the journal")
	}
	var types []string
	for range 2 {
		event := <-sub.Events()
		types = append(types, event.Type)
		if event.Partner != "a" {
			t.Fatalf("a subscriber to a heard %+v", event)
		}
	}
	if types[0] != "message.received" || types[1] != "ack.scheduled" {
		t.Fatalf("events %v", types)
	}
	replayed := broker.Subscribe(1, "")
	defer replayed.Close()
	if event := <-replayed.Events(); event.ID != 2 {
		t.Fatalf("replay after 1 began at %d", event.ID)
	}
}

func TestASnapshotRestoresTheJournal(t *testing.T) {
	t.Parallel()

	j := New(10, NewBroker(10, fixedNow), fixedNow)
	due := fixedNow().Add(time.Minute)
	j.Add("message.scheduled", &Message{Partner: "a", State: StateScheduled, DueAt: &due})
	j.Add("message.received", &Message{Partner: "a", State: StateReceived})
	snapshot := j.Snapshot()

	restored := New(10, NewBroker(10, fixedNow), fixedNow)
	restored.Restore(snapshot)
	if len(restored.Messages("a", Filter{})) != 2 {
		t.Fatal("the restored journal lost messages")
	}
	if pending := restored.Pending(); len(pending) != 1 || !pending[0].DueAt.Equal(due) {
		t.Fatalf("pending %+v", pending)
	}
	if restored.Events().Last() != snapshot.EventSeq {
		t.Fatalf(
			"event ids restart at %d, want after %d",
			restored.Events().Last(),
			snapshot.EventSeq,
		)
	}
	next := restored.Add("message.received", &Message{Partner: "a"})
	if next.ID != "a-000003" {
		t.Fatalf("the sequence restarted: %s", next.ID)
	}
}

func TestASlowSubscriberIsDropped(t *testing.T) {
	t.Parallel()

	broker := NewBroker(10, fixedNow)
	sub := broker.Subscribe(0, "")
	for range subscriberBacklog + 1 {
		broker.Publish("message.received", "a", nil)
	}
	count := 0
	for range sub.Events() {
		count++
	}
	if count != subscriberBacklog {
		t.Fatalf("read %d events before the stream closed", count)
	}
}
