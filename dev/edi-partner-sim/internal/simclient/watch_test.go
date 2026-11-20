package simclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
)

func event(w http.ResponseWriter, id int, partner, key string) {
	fmt.Fprintf(
		w,
		"id: %d\nevent: message.received\ndata: {\"id\":%d,\"type\":\"message.received\","+
			"\"partner\":%q,\"message\":{\"id\":\"%s-%06d\",\"partner\":%q,\"direction\":\"received\","+
			"\"purpose\":\"document\",\"state\":\"received\",\"sets\":[\"990\"],\"keys\":[%q],"+
			"\"raw\":\"\"}}\n\n",
		id,
		id,
		partner,
		partner,
		id,
		partner,
		key,
	)
	w.(http.Flusher).Flush()
}

func TestTheWatchResumesAfterAReconnect(t *testing.T) {
	t.Parallel()
	var connections atomic.Int32
	var resumedFrom atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		switch connections.Add(1) {
		case 1:
			event(w, 1, "acme", "K1")
		default:
			resumedFrom.Store(r.Header.Get("Last-Event-ID"))
			event(w, 2, "acme", "K2")
			<-r.Context().Done()
		}
	}))
	defer server.Close()
	watch, err := New(server.URL, nil).Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()
	err = watch.Until(context.Background(), time.Now().Add(10*time.Second), func(v View) bool {
		return len(v.About("acme", "K1")) == 1 && len(v.About("acme", "K2")) == 1
	})
	if err != nil {
		t.Fatalf("both events: %v", err)
	}
	if resumedFrom.Load() != "1" {
		t.Fatalf("the reconnect resumed from %v", resumedFrom.Load())
	}
}

func TestUntilEndsAtTheDeadline(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	watch, err := New(server.URL, nil).Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()
	started := time.Now()
	err = watch.Until(context.Background(), time.Now().Add(150*time.Millisecond),
		func(View) bool { return false })
	if !errors.Is(err, ErrDeadline) || time.Since(started) > 2*time.Second {
		t.Fatalf("a wait past its deadline: %v after %s", err, time.Since(started))
	}
	watch.Observe([]journal.Message{{ID: "acme-000009", Partner: "acme", Keys: []string{"K9"}}})
	watch.Read(func(v View) {
		if len(v.About("acme", "K9")) != 1 {
			t.Fatal("an observed message is not indexed by its key")
		}
	})
}

func TestAWatchOnADeadSimulatorFails(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	if _, err := New(server.URL, nil).Watch(context.Background()); err == nil {
		t.Fatal("watched a simulator with no event stream")
	}
}
