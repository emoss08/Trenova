package sim

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

type WebhookEvent struct {
	EventType string `json:"eventType"`
	EventTime string `json:"eventTime"`
	Data      any    `json:"data"`
}

type deliveryJob struct {
	Target           WebhookTarget
	Payload          []byte
	EventType        string
	Profile          string
	Identity         string
	DeliveryID       string
	DeliverySequence int
	InitialDelay     time.Duration
	TimestampSkew    time.Duration
	RetryJitter      time.Duration
}

type Dispatcher struct {
	cfg        config.WebhooksConfig
	store      *Store
	logger     *slog.Logger
	client     *http.Client
	jobs       chan *deliveryJob
	workerStop context.CancelFunc
	wg         sync.WaitGroup
	clock      *SimClock
	faults     *FaultEngine
	seq        atomic.Uint64
}

type webhookDeliveryOptions struct {
	AllowDuplicates    bool
	DuplicateRate      float64
	MaxDuplicates      int
	AllowReorder       bool
	ReorderMaxDelay    time.Duration
	AllowTimestampSkew bool
	TimestampSkewMax   time.Duration
	RetryJitterMax     time.Duration
}

func defaultWebhookDeliveryOptions() webhookDeliveryOptions {
	return webhookDeliveryOptions{
		AllowDuplicates:    true,
		DuplicateRate:      0.18,
		MaxDuplicates:      2,
		AllowReorder:       true,
		ReorderMaxDelay:    1200 * time.Millisecond,
		AllowTimestampSkew: true,
		TimestampSkewMax:   2 * time.Minute,
		RetryJitterMax:     350 * time.Millisecond,
	}
}

func NewDispatcher(
	cfg config.WebhooksConfig,
	store *Store,
	logger *slog.Logger,
) *Dispatcher {
	handler := logger
	if handler == nil {
		handler = slog.Default()
	}

	dispatcher := &Dispatcher{
		cfg:    cfg,
		store:  store,
		logger: handler,
		client: &http.Client{Timeout: 5 * time.Second},
		jobs:   make(chan *deliveryJob, 256),
	}
	dispatcher.startWorkers(2)
	return dispatcher
}

func (d *Dispatcher) Shutdown() {
	if d.workerStop != nil {
		d.workerStop()
	}
	d.wg.Wait()
}

func (d *Dispatcher) SetClock(clock *SimClock) {
	d.clock = clock
}

func (d *Dispatcher) SetFaultEngine(faults *FaultEngine) {
	d.faults = faults
}

func (d *Dispatcher) Dispatch(profile, eventType string, data any) error {
	if !d.cfg.Enabled {
		return nil
	}

	trimmedType := strings.TrimSpace(eventType)
	if trimmedType == "" {
		return ErrWebhookEventTypeRequired
	}

	targets := d.store.WebhookTargets(trimmedType)
	if len(targets) == 0 {
		return nil
	}

	event := WebhookEvent{
		EventType: trimmedType,
		EventTime: d.nowUTC().Format(time.RFC3339),
		Data:      data,
	}
	payload, err := sonic.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	for idx := range targets {
		target := targets[idx]
		options := deliveryOptionsForTarget(&target)
		identity := eventIdentity(data)
		duplicates := d.duplicateCount(options, &target, trimmedType, identity)
		totalDeliveries := 1 + duplicates
		if totalDeliveries < 1 {
			totalDeliveries = 1
		}

		for deliverySequence := 0; deliverySequence < totalDeliveries; deliverySequence++ {
			deliveryID := fmt.Sprintf("whdlv-%09d", d.seq.Add(1))
			delay, timestampSkew, retryJitter := d.deliveryModifiers(
				options,
				&target,
				trimmedType,
				identity,
				deliverySequence,
			)
			job := &deliveryJob{
				Target:           target,
				Payload:          append([]byte{}, payload...),
				EventType:        trimmedType,
				Profile:          strings.TrimSpace(profile),
				Identity:         identity,
				DeliveryID:       deliveryID,
				DeliverySequence: deliverySequence,
				InitialDelay:     delay,
				TimestampSkew:    timestampSkew,
				RetryJitter:      retryJitter,
			}
			select {
			case d.jobs <- job:
			default:
				return ErrWebhookQueueSaturated
			}
		}
	}

	return nil
}

func (d *Dispatcher) startWorkers(count int) {
	ctx, cancel := context.WithCancel(context.Background())
	d.workerStop = cancel

	for idx := 0; idx < count; idx++ {
		d.wg.Add(1)
		go d.worker(ctx)
	}
}

func (d *Dispatcher) worker(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-d.jobs:
			d.deliverWithRetry(ctx, job)
		}
	}
}

func (d *Dispatcher) deliverWithRetry(ctx context.Context, job *deliveryJob) {
	if job == nil {
		return
	}
	if job.InitialDelay > 0 {
		timer := time.NewTimer(job.InitialDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	attempt := 1
	backoff := d.cfg.InitialBackoff + job.RetryJitter
	if backoff < 50*time.Millisecond {
		backoff = 50 * time.Millisecond
	}
	for attempt <= d.cfg.MaxAttempts {
		err := d.sendOnce(ctx, job, attempt)
		if err == nil {
			return
		}

		if attempt >= d.cfg.MaxAttempts {
			d.logger.Warn(
				"webhook delivery failed",
				slog.String("webhookID", job.Target.ID),
				slog.String("url", job.Target.URL),
				slog.Int("attempts", attempt),
				slog.String("error", err.Error()),
			)
			return
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff *= 2
		attempt++
	}
}

func (d *Dispatcher) sendOnce(ctx context.Context, job *deliveryJob, attempt int) error {
	dropped, err := d.applyWebhookFault(ctx, job)
	if err != nil {
		return err
	}
	if dropped {
		return nil
	}

	urlValue := strings.TrimSpace(job.Target.URL)
	if urlValue == "" {
		return ErrWebhookURLRequired
	}

	requestCtx, cancel := context.WithTimeout(ctx, d.client.Timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		urlValue,
		bytes.NewReader(job.Payload),
	)
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Samsara-Sim-Delivery-Id", job.DeliveryID)
	request.Header.Set("X-Samsara-Sim-Delivery-Sequence", strconv.Itoa(job.DeliverySequence))
	request.Header.Set("X-Samsara-Sim-Delivery-Attempt", strconv.Itoa(attempt))

	signatureTime := d.nowUTC().Add(job.TimestampSkew)
	timestamp := strconv.FormatInt(signatureTime.Unix(), 10)
	signatureSecret := strings.TrimSpace(job.Target.Secret)
	if signatureSecret == "" {
		signatureSecret = strings.TrimSpace(d.cfg.SigningSecret)
	}
	if signatureSecret != "" {
		signature := signPayload(signatureSecret, timestamp, job.Payload)
		request.Header.Set("X-Samsara-Timestamp", timestamp)
		request.Header.Set("X-Samsara-Signature", signature)
	}

	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned status %d", response.StatusCode)
	}
	return nil
}

func (d *Dispatcher) applyWebhookFault(
	ctx context.Context,
	job *deliveryJob,
) (dropped bool, err error) {
	if d.faults == nil {
		return false, nil
	}

	signature := job.EventType + "|" + job.Identity + "|" + job.Target.ID
	decision, matched := d.faults.EvaluateWebhook(job.Profile, job.EventType, signature)
	if !matched {
		return false, nil
	}

	effect := decision.Rule.Effect
	if effect.LatencyMs > 0 {
		timer := time.NewTimer(time.Duration(effect.LatencyMs) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
	}
	if effect.Timeout {
		timer := time.NewTimer(d.client.Timeout + 2*time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
		return false, context.DeadlineExceeded
	}
	if effect.Drop {
		return true, nil
	}
	if effect.StatusCode != 0 {
		return false, fmt.Errorf("fault injected webhook status %d", effect.StatusCode)
	}
	return false, nil
}

func signPayload(secret, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func (d *Dispatcher) nowUTC() time.Time {
	if d.clock != nil {
		return d.clock.Now()
	}
	return time.Now().UTC()
}

func deliveryOptionsForTarget(target *WebhookTarget) webhookDeliveryOptions {
	options := defaultWebhookDeliveryOptions()
	if target == nil || len(target.SimDelivery) == 0 {
		return options
	}

	if raw, ok := target.SimDelivery["allowDuplicates"]; ok {
		options.AllowDuplicates = boolFromAny(raw, options.AllowDuplicates)
	}
	if raw, ok := target.SimDelivery["duplicateRate"]; ok {
		options.DuplicateRate = clampFloat64(floatFromAny(raw), 0, 1)
	}
	if raw, ok := target.SimDelivery["maxDuplicates"]; ok {
		options.MaxDuplicates = clampInt(int(intFromAny(raw)), 0, 4)
	}
	if raw, ok := target.SimDelivery["allowReorder"]; ok {
		options.AllowReorder = boolFromAny(raw, options.AllowReorder)
	}
	if raw, ok := target.SimDelivery["reorderWindowMs"]; ok {
		ms := clampInt64(intFromAny(raw), 0, 20_000)
		options.ReorderMaxDelay = time.Duration(ms) * time.Millisecond
	}
	if raw, ok := target.SimDelivery["allowTimestampSkew"]; ok {
		options.AllowTimestampSkew = boolFromAny(raw, options.AllowTimestampSkew)
	}
	if raw, ok := target.SimDelivery["timestampSkewSeconds"]; ok {
		seconds := clampInt64(intFromAny(raw), 0, 900)
		options.TimestampSkewMax = time.Duration(seconds) * time.Second
	}
	if raw, ok := target.SimDelivery["retryJitterMs"]; ok {
		ms := clampInt64(intFromAny(raw), 0, 5000)
		options.RetryJitterMax = time.Duration(ms) * time.Millisecond
	}
	return options
}

func (d *Dispatcher) duplicateCount(
	options webhookDeliveryOptions,
	target *WebhookTarget,
	eventType string,
	identity string,
) int {
	if !options.AllowDuplicates || options.MaxDuplicates <= 0 || options.DuplicateRate <= 0 {
		return 0
	}

	base := webhookDeterministicKey(target, eventType, identity)
	roll := d.hashFraction(base, "duplicate-roll")
	if roll >= options.DuplicateRate {
		return 0
	}
	if options.MaxDuplicates == 1 {
		return 1
	}
	rangeMax := options.MaxDuplicates
	if rangeMax < 1 {
		rangeMax = 1
	}
	count := 1 + int(math.Floor(float64(rangeMax)*d.hashFraction(base, "duplicate-count")))
	return clampInt(count, 1, options.MaxDuplicates)
}

func (d *Dispatcher) deliveryModifiers(
	options webhookDeliveryOptions,
	target *WebhookTarget,
	eventType string,
	identity string,
	deliverySequence int,
) (delay, timestampSkew, retryJitter time.Duration) {
	base := webhookDeterministicKey(target, eventType, identity)
	sequenceKey := strconv.Itoa(deliverySequence)

	if options.AllowReorder && options.ReorderMaxDelay > 0 {
		reorderFraction := d.hashFraction(base, "reorder-delay", sequenceKey)
		delay = time.Duration(float64(options.ReorderMaxDelay) * reorderFraction)
		if deliverySequence > 0 {
			delay += time.Duration(deliverySequence) * (options.ReorderMaxDelay / 6)
		}
		// Occasionally let duplicates race ahead to produce out-of-order delivery.
		raceRoll := d.hashFraction(base, "duplicate-race", sequenceKey)
		if deliverySequence > 0 && raceRoll < 0.22 {
			delay /= 2
		}
	}

	if options.AllowTimestampSkew && options.TimestampSkewMax > 0 {
		skewRoll := d.hashFraction(base, "timestamp-skew", sequenceKey)
		normalized := (2 * skewRoll) - 1
		timestampSkew = time.Duration(normalized * float64(options.TimestampSkewMax))
	}

	if options.RetryJitterMax > 0 {
		jitterRoll := d.hashFraction(base, "retry-jitter", sequenceKey)
		normalized := (2 * jitterRoll) - 1
		retryJitter = time.Duration(normalized * float64(options.RetryJitterMax))
	}
	return delay, timestampSkew, retryJitter
}

func (d *Dispatcher) hashFraction(parts ...string) float64 {
	hasher := fnv.New64a()
	for _, part := range parts {
		_, _ = hasher.Write([]byte(strings.TrimSpace(part)))
		_, _ = hasher.Write([]byte("|"))
	}
	value := hasher.Sum64() % 10000
	return float64(value) / 10000.0
}

func webhookDeterministicKey(target *WebhookTarget, eventType, identity string) string {
	targetID := ""
	targetURL := ""
	if target != nil {
		targetID = target.ID
		targetURL = target.URL
	}
	return strings.Join([]string{
		strings.TrimSpace(targetID),
		strings.TrimSpace(targetURL),
		strings.TrimSpace(eventType),
		strings.TrimSpace(identity),
	}, "|")
}

func boolFromAny(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return fallback
}

func intFromAny(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case int32:
		return int64(typed)
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	default:
		return 0
	}
}

func clampInt(value, lower, upper int) int {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}
