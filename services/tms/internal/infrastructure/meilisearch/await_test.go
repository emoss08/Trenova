package meilisearch

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func freeAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

func serveHealthLater(t *testing.T, addr string, after time.Duration) {
	t.Helper()

	server := &http.Server{
		ReadHeaderTimeout: time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"available"}`))
		}),
	}
	started := make(chan struct{})
	go func() {
		defer close(started)
		time.Sleep(after)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return
		}
		go func() { _ = server.Serve(l) }()
	}()

	t.Cleanup(func() {
		<-started
		_ = server.Close()
	})
}

func awaitParams(
	t *testing.T,
	addr string,
	wait time.Duration,
) (AwaitParams, *observer.ObservedLogs) {
	t.Helper()

	search := &searchconfig.Config{
		Enabled: true,
		Meilisearch: searchconfig.MeilisearchConfig{
			URL:     "http://" + addr,
			Timeout: time.Second,
		},
	}
	core, logs := observer.New(zap.InfoLevel)

	client, err := New(Params{
		SearchConfig: search,
		Logger:       zap.New(core),
		LC:           fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)

	return AwaitParams{
		Client:       client,
		SearchConfig: search,
		AppConfig:    &appconfig.Config{StartupWait: wait},
	}, logs
}

func TestAwaitReady_StartsWhenMeilisearchComesUpLate(t *testing.T) {
	addr := freeAddress(t)
	p, logs := awaitParams(t, addr, time.Hour)
	serveHealthLater(t, addr, 700*time.Millisecond)

	require.NoError(t, AwaitReady(p))

	assert.Equal(
		t,
		1,
		logs.FilterMessage("Waiting for Meilisearch to answer before starting").Len(),
	)
	assert.Equal(t, 1, logs.FilterMessage("Meilisearch answered; starting").Len())
	assert.Equal(t, 1, logs.FilterMessage("Meilisearch connection established").Len())
}

func TestAwaitReady_FailsOnceTheStartUpWaitIsSpent(t *testing.T) {
	p, _ := awaitParams(t, freeAddress(t), 0)

	err := AwaitReady(p)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Meilisearch did not answer within the start-up wait")
}

func TestAwaitReady_DisabledSearchWaitsForNothing(t *testing.T) {
	client, err := New(Params{
		SearchConfig: &searchconfig.Config{},
		Logger:       zap.NewNop(),
		LC:           fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)

	require.NoError(t, AwaitReady(AwaitParams{
		Client:       client,
		SearchConfig: &searchconfig.Config{},
		AppConfig:    &appconfig.Config{},
	}))
}
