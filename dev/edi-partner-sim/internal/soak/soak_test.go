package soak

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenovatest"
)

func serveSim(t *testing.T) string {
	t.Helper()
	definition, err := partner.Open("../play/testdata/partners/acme")
	if err != nil {
		t.Fatal(err)
	}
	var handler http.Handler = http.NotFoundHandler()
	var mu sync.RWMutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		current := handler
		mu.RUnlock()
		current.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	simulator, err := sim.New(sim.Options{
		Partners:    []*partner.Definition{definition},
		IdentityDir: t.TempDir(),
		PublicURL:   server.URL,
		Capacity:    1000,
		Seed:        1,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	handler = simulator.Handler()
	mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan struct{})
	go func() {
		_ = simulator.Run(ctx)
		close(ran)
	}()
	t.Cleanup(func() {
		cancel()
		<-ran
	})
	return server.URL
}

func TestSoakMeasuresEveryConversation(t *testing.T) {
	simURL := serveSim(t)
	fake := trenovatest.New(t, trenovatest.Envelope{
		UsQualifier: "ZZ", UsID: "TRENOVA", UsGroup: "TRENOVA",
		ThemQualifier: "ZZ", ThemID: "ACMESHIP", ThemGroup: "ACMESHIP", SCAC: "TRNV",
	})
	out := new(bytes.Buffer)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	report, err := Run(ctx, &Options{
		Play: play.Options{
			API: fake.API(), Sim: simURL, Email: "admin@trenova.app", Password: "admin123!",
			Root: "../play/testdata/partners", Run: 11, IdentityDir: t.TempDir(), Out: out,
		},
		Conversations: 6,
		Rate:          50,
		Concurrency:   3,
		Within:        15 * time.Second,
		Settle:        5 * time.Second,
	})
	if err != nil {
		t.Fatalf("soak: %v\n%s", err, out)
	}
	var summary bytes.Buffer
	report.WriteSummary(&summary)
	acme := report.Partners[0]
	if !report.Passed() || acme.Completed != 6 || acme.OpenFlows != 0 {
		t.Fatalf("soak failed:\n%s\n%s", summary.String(), out)
	}
	for _, name := range milestoneOrder {
		stats := acme.Milestones[name]
		if stats.Count != 6 || stats.P50 > stats.P90 || stats.P90 > stats.Max {
			t.Fatalf("milestone %s: %+v", name, stats)
		}
	}
	if !strings.Contains(summary.String(), "to 210") ||
		!strings.Contains(summary.String(), "PASS") {
		t.Fatalf("the summary:\n%s", summary.String())
	}
}

func TestPercentiles(t *testing.T) {
	t.Parallel()
	var samples []time.Duration
	for n := 1; n <= 100; n++ {
		samples = append(samples, time.Duration(n)*time.Second)
	}
	stats := statsOf(samples)
	if stats.Count != 100 || time.Duration(stats.P50) != 50*time.Second ||
		time.Duration(stats.P90) != 90*time.Second ||
		time.Duration(stats.P99) != 99*time.Second ||
		time.Duration(stats.Max) != 100*time.Second {
		t.Fatalf("stats: %+v", stats)
	}
	if one := statsOf([]time.Duration{time.Second}); time.Duration(one.P99) != time.Second {
		t.Fatalf("one sample: %+v", one)
	}
}
