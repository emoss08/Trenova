package retrainingalert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/webhooksig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.RetrainingAlerter = (*Webhook)(nil)

const (
	maxAttempts        = 3
	firstBackoff       = time.Second
	maxResponseBytes   = 4096
	userAgent          = "Trenova-Retraining-Alerts/1"
	headerID           = "webhook-id"
	headerTimestamp    = "webhook-timestamp"
	headerSignature    = "webhook-signature"
	headerEvent        = "X-Trenova-Event"
	contentTypeJSON    = "application/json"
	statusClassSuccess = 2
	statusClassClient  = 4
)

var (
	errDeliveryRejected = errors.New("the alert endpoint rejected the delivery")
	errDeliveryFailed   = errors.New("the alert endpoint did not accept the delivery")
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Webhook struct {
	cfg        *cloudconfig.AIRetrainingAlertsConfig
	instanceID string
	client     *http.Client
	l          *zap.Logger
	now        func() time.Time
	sleep      func(ctx context.Context, d time.Duration) error
}

func New(p Params) *Webhook {
	alerts := &cloudconfig.From(p.Config).AIRetraining.Alerts

	return &Webhook{
		cfg:        alerts,
		instanceID: p.Config.Platform.InstanceID,
		client: httpsafe.NewClientWithPolicy(
			alerts.GetTimeout(),
			httpsafe.Policy{AllowPrivateNetworks: alerts.AllowPrivateNetwork},
		),
		l:     p.Logger.Named("infrastructure.retraining-alert"),
		now:   time.Now,
		sleep: sleepContext,
	}
}

func AsAlerter(w *Webhook) services.RetrainingAlerter { return w }

type cyclePayload struct {
	ID                   string  `json:"id"`
	Status               string  `json:"status"`
	Trigger              string  `json:"trigger"`
	SkipReason           string  `json:"skipReason,omitempty"`
	RequestedBy          string  `json:"requestedBy"`
	ExportID             string  `json:"exportId,omitempty"`
	NewExamples          int     `json:"newExamples"`
	MinNewExamples       int     `json:"minNewExamples"`
	DriftingProviders    int     `json:"driftingProviders"`
	Trainer              string  `json:"trainer,omitempty"`
	Attempts             int     `json:"attempts"`
	TrainingConfig       string  `json:"trainingConfig,omitempty"`
	ModelDirectory       string  `json:"modelDirectory,omitempty"`
	Examples             int     `json:"examples"`
	ModelAccuracy        float64 `json:"modelAccuracy"`
	BaselineAccuracy     float64 `json:"baselineAccuracy"`
	MinAccuracyPercent   int     `json:"minAccuracyPercent"`
	MaxRegressionPoints  int     `json:"maxRegressionPoints"`
	GateMessage          string  `json:"gateMessage,omitempty"`
	FailureMessage       string  `json:"failureMessage,omitempty"`
	StructuredOutputMode string  `json:"structuredOutputMode"`
	CreatedAt            int64   `json:"createdAt"`
	FinishedAt           *int64  `json:"finishedAt,omitempty"`
}

type payload struct {
	Text       string       `json:"text"`
	Event      string       `json:"event"`
	InstanceID string       `json:"instanceId,omitempty"`
	SentAt     int64        `json:"sentAt"`
	Cycle      cyclePayload `json:"cycle"`
}

func newPayload(cycle *aitraining.RetrainingCycle, instanceID string, sentAt int64) *payload {
	out := &payload{
		Text:       cycle.AlertSummary(),
		Event:      cycle.Status.EventName(),
		InstanceID: instanceID,
		SentAt:     sentAt,
		Cycle: cyclePayload{
			ID:                   cycle.ID.String(),
			Status:               cycle.Status.String(),
			Trigger:              cycle.Trigger.String(),
			SkipReason:           cycle.SkipReason.String(),
			RequestedBy:          cycle.RequestedBy,
			NewExamples:          cycle.NewExamples,
			MinNewExamples:       cycle.MinNewExamples,
			DriftingProviders:    cycle.DriftingProviders,
			Trainer:              cycle.Trainer,
			Attempts:             cycle.Attempts,
			TrainingConfig:       cycle.TrainingConfig,
			ModelDirectory:       cycle.ModelDirectory,
			Examples:             cycle.Examples,
			ModelAccuracy:        cycle.ModelAccuracy(),
			BaselineAccuracy:     cycle.BaselineAccuracy(),
			MinAccuracyPercent:   cycle.MinAccuracyPercent,
			MaxRegressionPoints:  cycle.MaxRegressionPoints,
			GateMessage:          cycle.GateMessage,
			FailureMessage:       cycle.FailureMessage,
			StructuredOutputMode: string(cycle.StructuredOutputMode),
			CreatedAt:            cycle.CreatedAt,
			FinishedAt:           cycle.FinishedAt,
		},
	}
	if cycle.ExportID != nil {
		out.Cycle.ExportID = cycle.ExportID.String()
	}

	return out
}

func (w *Webhook) AlertRetraining(ctx context.Context, cycle *aitraining.RetrainingCycle) error {
	if !w.cfg.Enabled() || cycle == nil {
		return nil
	}

	now := w.now()
	body, err := sonic.Marshal(newPayload(cycle, w.instanceID, now.Unix()))
	if err != nil {
		return fmt.Errorf("encode retraining alert: %w", err)
	}
	deliveryID := cycle.ID.String() + ":" + strings.ToLower(cycle.Status.String())

	backoff := firstBackoff
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retry, sendErr := w.send(ctx, deliveryID, cycle.Status.EventName(), body)
		if sendErr == nil {
			return nil
		}
		lastErr = sendErr
		if !retry || attempt == maxAttempts {
			break
		}
		if err = w.sleep(ctx, backoff); err != nil {
			return errors.Join(lastErr, err)
		}
		backoff *= 2
	}

	return fmt.Errorf("deliver retraining alert %s: %w", deliveryID, lastErr)
}

func (w *Webhook) send(
	ctx context.Context,
	deliveryID, event string,
	body []byte,
) (retry bool, err error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimSpace(w.cfg.WebhookURL),
		bytes.NewReader(body),
	)
	if err != nil {
		return false, fmt.Errorf("build retraining alert request: %w", err)
	}
	timestamp := strconv.FormatInt(w.now().Unix(), 10)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(headerEvent, event)
	req.Header.Set(headerID, deliveryID)
	req.Header.Set(headerTimestamp, timestamp)
	if w.cfg.Signed() {
		signature, signErr := webhooksig.SignSvix(w.cfg.Secret, deliveryID, timestamp, body)
		if signErr != nil {
			return false, fmt.Errorf("sign retraining alert: %w", signErr)
		}
		req.Header.Set(headerSignature, signature)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("send retraining alert: %w", err)
	}
	defer resp.Body.Close()
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))

	switch resp.StatusCode / 100 {
	case statusClassSuccess:
		return false, nil
	case statusClassClient:
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusRequestTimeout {
			return true, fmt.Errorf("%w: %s", errDeliveryFailed, resp.Status)
		}
		w.l.Warn("retraining alert rejected",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("response", detail),
		)
		return false, fmt.Errorf("%w: %s", errDeliveryRejected, resp.Status)
	default:
		return true, fmt.Errorf("%w: %s", errDeliveryFailed, resp.Status)
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
