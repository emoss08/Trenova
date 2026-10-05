package retrainingalert

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var testSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

type delivery struct {
	header http.Header
	body   []byte
}

type receiver struct {
	mu         sync.Mutex
	deliveries []delivery
	statuses   []int
}

func (r *receiver) handler(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliveries = append(r.deliveries, delivery{header: req.Header.Clone(), body: body})
	status := http.StatusNoContent
	if len(r.statuses) > 0 {
		status = r.statuses[0]
		r.statuses = r.statuses[1:]
	}
	w.WriteHeader(status)
}

func (r *receiver) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.deliveries)
}

type fixture struct {
	webhook  *Webhook
	receiver *receiver
	sleeps   []time.Duration
	now      time.Time
}

func newFixture(t *testing.T, alerts cloudconfig.AIRetrainingAlertsConfig, statuses ...int) *fixture {
	t.Helper()

	rec := &receiver{statuses: statuses}
	server := httptest.NewServer(http.HandlerFunc(rec.handler))
	t.Cleanup(server.Close)

	if alerts.WebhookURL == "" {
		alerts.WebhookURL = server.URL + "/hooks/retraining"
	}
	cfg := cloudconfig.WithSettings(&config.Config{}, &cloudconfig.Settings{AIRetraining: cloudconfig.AIRetrainingConfig{Alerts: alerts}})
	cfg.Platform.InstanceID = "trenova-prod"

	f := &fixture{receiver: rec, now: time.Unix(1_790_000_000, 0)}
	f.webhook = New(Params{Config: cfg, Logger: zap.NewNop()})
	f.webhook.now = func() time.Time { return f.now }
	f.webhook.sleep = func(_ context.Context, d time.Duration) error {
		f.sleeps = append(f.sleeps, d)
		return nil
	}
	return f
}

func passedCycle() *aitraining.RetrainingCycle {
	exportID := pulid.MustNew("aitx_")
	finished := int64(1_790_000_500)
	return &aitraining.RetrainingCycle{
		ID:                   pulid.MustNew("airc_"),
		Status:               aitraining.RetrainingStatusPassed,
		Trigger:              aitraining.RetrainingTriggerDrift,
		RequestedBy:          aitraining.RetrainingRequestedByScheduler,
		ExportID:             &exportID,
		NewExamples:          1400,
		MinNewExamples:       1000,
		Trainer:              "gpu-1",
		Attempts:             1,
		ModelDirectory:       "/data/retraining/run/dpo/model",
		Examples:             120,
		ModelCorrect:         912,
		ModelScored:          1000,
		BaselineCorrect:      890,
		BaselineScored:       1000,
		MinAccuracyPercent:   85,
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		CreatedAt:            1_790_000_000,
		FinishedAt:           &finished,
	}
}

func TestAlertDoesNothingWithoutAWebhook(t *testing.T) {
	t.Parallel()

	w := New(Params{Config: &config.Config{}, Logger: zap.NewNop()})
	require.NoError(t, w.AlertRetraining(t.Context(), passedCycle()))
}

func TestAlertSendsASignedSummary(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{Secret: testSecret, AllowPrivateNetwork: true})
	cycle := passedCycle()

	require.NoError(t, f.webhook.AlertRetraining(t.Context(), cycle))
	require.Equal(t, 1, f.receiver.count())

	got := f.receiver.deliveries[0]
	deliveryID := cycle.ID.String() + ":passed"
	assert.Equal(t, deliveryID, got.header.Get("webhook-id"))
	assert.Equal(t, "retraining.passed", got.header.Get("X-Trenova-Event"))
	assert.Equal(t, "application/json", got.header.Get("Content-Type"))
	require.NoError(t, webhooksig.VerifySvix(webhooksig.SvixParams{
		Secret:    testSecret,
		ID:        got.header.Get("webhook-id"),
		Timestamp: got.header.Get("webhook-timestamp"),
		Signature: got.header.Get("webhook-signature"),
		Body:      got.body,
		Now:       f.now,
	}), "a receiver can verify the delivery with the shared secret")

	var decoded payload
	require.NoError(t, sonic.Unmarshal(got.body, &decoded))
	assert.Equal(t, "retraining.passed", decoded.Event)
	assert.Equal(t, "trenova-prod", decoded.InstanceID)
	assert.Contains(t, decoded.Text, "passed: model 91.20% against production 89.00%")
	assert.Equal(t, cycle.ID.String(), decoded.Cycle.ID)
	assert.Equal(t, cycle.ExportID.String(), decoded.Cycle.ExportID)
	assert.InDelta(t, 0.912, decoded.Cycle.ModelAccuracy, 1e-9)
	assert.Equal(t, "/data/retraining/run/dpo/model", decoded.Cycle.ModelDirectory)
}

func TestAlertWithoutASecretIsUnsigned(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{AllowPrivateNetwork: true})
	require.NoError(t, f.webhook.AlertRetraining(t.Context(), passedCycle()))
	assert.Empty(t, f.receiver.deliveries[0].header.Get("webhook-signature"))
}

func TestAlertRetriesServerErrorsWithBackoff(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{AllowPrivateNetwork: true},
		http.StatusBadGateway, http.StatusTooManyRequests, http.StatusOK)

	require.NoError(t, f.webhook.AlertRetraining(t.Context(), passedCycle()))
	assert.Equal(t, 3, f.receiver.count())
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, f.sleeps)
	assert.Equal(t, f.receiver.deliveries[0].header.Get("webhook-id"),
		f.receiver.deliveries[2].header.Get("webhook-id"),
		"a retry keeps the delivery id, so a receiver can drop duplicates")
}

func TestAlertGivesUpAfterThreeAttempts(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{AllowPrivateNetwork: true},
		http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusServiceUnavailable)

	err := f.webhook.AlertRetraining(t.Context(), passedCycle())
	require.ErrorIs(t, err, errDeliveryFailed)
	assert.Equal(t, 3, f.receiver.count())
}

func TestAlertDoesNotRetryARejection(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{AllowPrivateNetwork: true}, http.StatusUnauthorized)

	err := f.webhook.AlertRetraining(t.Context(), passedCycle())
	require.ErrorIs(t, err, errDeliveryRejected)
	assert.Equal(t, 1, f.receiver.count())
	assert.Empty(t, f.sleeps)
}

func TestAlertStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{AllowPrivateNetwork: true}, http.StatusBadGateway)
	errStopped := errors.New("stopped")
	f.webhook.sleep = func(context.Context, time.Duration) error { return errStopped }

	err := f.webhook.AlertRetraining(t.Context(), passedCycle())
	require.ErrorIs(t, err, errStopped)
	assert.Equal(t, 1, f.receiver.count())
}

func TestAlertRefusesAPrivateAddressUnlessAllowed(t *testing.T) {
	t.Parallel()

	f := newFixture(t, cloudconfig.AIRetrainingAlertsConfig{})

	err := f.webhook.AlertRetraining(t.Context(), passedCycle())
	require.Error(t, err)
	assert.Zero(t, f.receiver.count(), "a loopback endpoint is refused without allowPrivateNetwork")
	assert.True(t, strings.Contains(err.Error(), "disallowed network"), err.Error())
}
