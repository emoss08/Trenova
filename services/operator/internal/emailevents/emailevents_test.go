package emailevents_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

const (
	token     = "managed-token"
	recipient = "dispatcher@acme.example"
)

type staticTokens struct{}

func (staticTokens) Token(context.Context) (string, error)   { return token, nil }
func (staticTokens) Refresh(context.Context) (string, error) { return token, nil }

type rig struct {
	cp    *controlplanetest.Server
	tms   *tmstest.Server
	relay *emailevents.Relay
	logs  *bytes.Buffer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
	bundle := cp.Bundle(time.Hour)
	ident, err := bundle.Identity()
	require.NoError(t, err)
	tr := &credential.Transport{}
	require.NoError(t, tr.Use(&ident))
	api, err := controlplane.New(cp.URL, &http.Client{Transport: tr})
	require.NoError(t, err)

	srv := tmstest.New(t, token)
	client, err := tms.New(tms.Params{URL: srv.URL, Tokens: staticTokens{}})
	require.NoError(t, err)

	logs := &bytes.Buffer{}
	relay, err := emailevents.New(&emailevents.Params{
		ControlPlane: api,
		TMS:          client,
		Logger:       slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	require.NoError(t, err)
	return &rig{cp: cp, tms: srv, relay: relay, logs: logs}
}

func (r *rig) queue(n int) []uuid.UUID {
	ids := make([]uuid.UUID, 0, n)
	for range n {
		id := uuid.New()
		ids = append(ids, id)
		r.cp.QueueEmailEvent(&instanceapi.EmailEvent{
			Id:       id,
			Provider: instanceapi.EmailEventPostmark,
			Payload: `{"RecordType":"Bounce","ID":883953,"Type":"HardBounce",` +
				`"MessageID":"pm-1","Email":"` + recipient + `"}`,
			ReceivedAt: time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC),
		})
	}
	return ids
}

func TestARelayNeedsItsDependencies(t *testing.T) {
	t.Parallel()
	_, err := emailevents.New(&emailevents.Params{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the control plane")
	assert.Contains(t, err.Error(), "the TMS")
}

func TestEventsAreDeliveredToTheTMSThenAcknowledged(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ids := r.queue(2)

	res, err := r.relay.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, emailevents.Result{Listed: 2, Handled: 2}, res)
	delivered := r.tms.EmailEvents()
	require.Len(t, delivered, 2)
	assert.Equal(t, ids[0].String(), delivered[0].ID)
	assert.Equal(t, "Postmark", delivered[0].Provider)
	assert.Contains(t, delivered[0].Payload, "HardBounce", "the payload reaches the TMS verbatim")
	assert.Equal(t, [][]uuid.UUID{ids}, r.cp.EmailEventAcks())
	assert.Empty(t, r.cp.QueuedEmailEvents())
	assert.NotContains(t, r.logs.String(), recipient)
}

func TestOnlyTheEventsTheTMSHandledAreAcknowledged(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ids := r.queue(3)
	r.tms.LeaveEmailEventsUnhandled(ids[1].String())

	res, err := r.relay.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, emailevents.Result{Listed: 3, Handled: 2}, res)
	assert.Equal(t, [][]uuid.UUID{{ids[0], ids[2]}}, r.cp.EmailEventAcks())
	queued := r.cp.QueuedEmailEvents()
	require.Len(t, queued, 1)
	assert.Equal(t, ids[1], queued[0].Id)
	assert.Contains(t, r.logs.String(), ids[1].String(), "the log names the event left to retry")
	assert.NotContains(t, r.logs.String(), recipient)

	r.tms.HandleEmailEvents(ids[1].String())
	_, err = r.relay.Run(t.Context())
	require.NoError(t, err)
	assert.Empty(t, r.cp.QueuedEmailEvents(), "the next run delivers it again")
}

func TestATMSFailureAcknowledgesNothingAndTheNextRunRetries(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ids := r.queue(2)
	r.tms.FailEmailEventsNext(1)

	_, err := r.relay.Run(t.Context())

	require.Error(t, err)
	assert.Empty(t, r.cp.EmailEventAcks())
	assert.Len(t, r.cp.QueuedEmailEvents(), 2)
	assert.NotContains(t, err.Error(), recipient)
	assert.NotContains(t, r.logs.String(), recipient)

	res, err := r.relay.Run(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, res.Handled)
	assert.Equal(t, [][]uuid.UUID{ids}, r.cp.EmailEventAcks())
}

func TestAtMostOneBatchIsRelayedARun(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.queue(emailevents.MaxBatch + 5)

	res, err := r.relay.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, emailevents.MaxBatch, res.Listed)
	assert.Len(t, r.tms.EmailEvents(), emailevents.MaxBatch)
	assert.Len(t, r.cp.QueuedEmailEvents(), 5)
}

func TestNothingWaitingSendsNothingToTheTMS(t *testing.T) {
	t.Parallel()
	r := newRig(t)

	res, err := r.relay.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, emailevents.Result{}, res)
	assert.Empty(t, r.tms.EmailEvents())
	assert.Empty(t, r.cp.EmailEventAcks())
}
