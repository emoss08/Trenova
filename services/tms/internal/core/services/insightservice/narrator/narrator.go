// Package narrator turns a detector's findings into prose a person wants to read.
//
// It is the only place in the insight pipeline that talks to a model, and it is
// deliberately the least trusted. Everything it produces is wording: it cannot
// change a number, choose a severity, add a link, or decide that a finding
// matters. If it fails, is switched off, or returns something that cites a
// figure nobody computed, the caller keeps the detector's own headline and the
// insight is complete without it.
package narrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxOutputTokens bounds one narration. Three short fields cannot need more,
	// and an unbounded ceiling on a per-finding call is how a refresh quietly
	// becomes expensive.
	maxOutputTokens = 700
	// maxFindingsPerCall is how many findings are described in one request.
	// Batching matters: a hundred findings narrated one at a time is a hundred
	// round trips, and the model also writes better copy when it can see that
	// three of them are the same customer.
	maxFindingsPerCall = 8
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Completion services.CompletionService
}

type Service struct {
	l          *zap.Logger
	completion services.CompletionService
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.insight-narrator"),
		completion: p.Completion,
	}
}

// Narration is the wording for one finding.
type Narration struct {
	Headline       string
	Narrative      string
	Recommendation string
	// Narrated is false when the model was unavailable, said nothing about this
	// finding, or was overruled by the number guard. The caller keeps the
	// detector's own wording in that case, and the insight records that no model
	// wrote it.
	Narrated        bool
	ModelIdentifier string
	ProviderID      pulid.ID
}

// NarrateRequest is one batch of findings to describe.
type NarrateRequest struct {
	TenantInfo pagination.TenantInfo
	Findings   []detector.Finding
	WindowDays int
}

// Narrate writes prose for each finding, keyed by dedupe key.
//
// Every finding is represented in the result whether or not the model produced
// anything for it, so a caller never has to distinguish "no narration" from
// "finding dropped". A finding the model skipped comes back with the detector's
// own headline and Narrated false.
func (s *Service) Narrate(
	ctx context.Context,
	req *NarrateRequest,
) map[string]Narration {
	result := make(map[string]Narration, len(req.Findings))
	for _, finding := range req.Findings {
		result[finding.DedupeKey] = fallbackFor(finding)
	}

	for batch := range chunk(req.Findings, maxFindingsPerCall) {
		s.narrateBatch(ctx, req, batch, result)
	}

	return result
}

func (s *Service) narrateBatch(
	ctx context.Context,
	req *NarrateRequest,
	batch []detector.Finding,
	into map[string]Narration,
) {
	completion, err := s.completion.CompleteStructured(ctx, &services.StructuredCompletionRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aiprovider.TaskOperationalInsights,
		System:       systemPrompt,
		Context:      buildContext(batch, req.WindowDays),
		OutputSchema: outputSchema(),
		SchemaName:   "operational_insights",
		MaxTokens:    maxOutputTokens,
	})
	if err != nil {
		// An unconfigured or failing provider is not an error for the caller: the
		// findings are already complete and keep their own wording. It is logged at
		// info for the unconfigured case because that is a deployment choice rather
		// than a fault.
		s.logNarrationUnavailable(err, len(batch))

		return
	}

	narrations, err := parseNarrations(completion.Text)
	if err != nil {
		s.l.Warn("insight narration could not be parsed",
			zap.String("model", completion.ModelIdentifier),
			zap.Error(err),
		)

		return
	}

	s.applyNarrations(applyParams{
		batch:      batch,
		narrations: narrations,
		completion: completion,
		windowDays: req.WindowDays,
		into:       into,
	})
}

type applyParams struct {
	batch      []detector.Finding
	narrations map[string]narratedFinding
	completion *services.StructuredCompletionResult
	windowDays int
	into       map[string]Narration
}

// applyNarrations accepts a model's wording only where it survives the guard.
//
// The check runs per finding rather than per batch so one fabricated figure
// costs that finding its prose and no other. Every rejection is logged with what
// the model actually claimed, because a model that keeps inventing numbers is a
// provider configuration problem someone needs to see.
func (s *Service) applyNarrations(params applyParams) {
	for _, finding := range params.batch {
		narrated, ok := params.narrations[finding.DedupeKey]
		if !ok {
			continue
		}

		supported := supportedFor(finding, params.windowDays)
		prose := strings.Join(
			[]string{narrated.Headline, narrated.Narrative, narrated.Recommendation},
			" ",
		)

		if check := numberguard.CheckNumbers(prose, supported); !check.OK {
			s.l.Warn("rejected insight narration citing figures no detector computed",
				zap.String("detector", finding.DedupeKey),
				zap.String("model", params.completion.ModelIdentifier),
				zap.Strings("unsupported", check.Unsupported),
			)

			continue
		}

		params.into[finding.DedupeKey] = Narration{
			Headline: firstNonEmpty(
				trim(narrated.Headline, insight.MaxHeadlineLength),
				finding.Headline,
			),
			Narrative:       trim(narrated.Narrative, insight.MaxNarrativeLength),
			Recommendation:  trim(narrated.Recommendation, insight.MaxRecommendationLength),
			Narrated:        true,
			ModelIdentifier: params.completion.ModelIdentifier,
			ProviderID:      params.completion.ProviderID,
		}
	}
}

func (s *Service) logNarrationUnavailable(err error, findings int) {
	if errorsIsNoProvider(err) {
		s.l.Info("no provider configured for insight narration; using detector wording",
			zap.Int("findings", findings),
		)

		return
	}

	s.l.Warn("insight narration failed; using detector wording",
		zap.Int("findings", findings),
		zap.Error(err),
	)
}

// fallbackFor is what an insight says when no model wrote it: the detector's own
// headline, which is plainer and always true.
func fallbackFor(finding detector.Finding) Narration {
	return Narration{Headline: finding.Headline}
}

// supportedFor is every figure this finding's prose may legitimately cite.
func supportedFor(finding detector.Finding, windowDays int) []decimal.Decimal {
	metricValues := make([]decimal.Decimal, 0, len(finding.Metrics))
	baselines := make([]decimal.Decimal, 0, len(finding.Metrics))

	for _, metric := range finding.Metrics {
		metricValues = append(metricValues, metric.Value)
		if metric.Baseline != nil {
			baselines = append(baselines, *metric.Baseline)
		} else {
			// Kept positionally aligned with metricValues so the value-versus-
			// baseline difference is computed against the right pair.
			baselines = append(baselines, metric.Value)
		}
	}

	// The window length is a number the prose is encouraged to state and the
	// detector did not put in a metric, so it is vouched for explicitly.
	return numberguard.SupportedValues(
		metricValues,
		baselines,
		decimal.NewFromInt(int64(windowDays)),
		decimal.NewFromInt(int64(len(finding.Links))),
	)
}

func trim(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}

	return strings.TrimSpace(string(runes[:limit]))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

// chunk yields successive slices of at most size elements.
func chunk[T any](items []T, size int) func(func([]T) bool) {
	return func(yield func([]T) bool) {
		for start := 0; start < len(items); start += size {
			end := min(start+size, len(items))
			if !yield(items[start:end]) {
				return
			}
		}
	}
}

func parseNarrations(payload string) (map[string]narratedFinding, error) {
	var envelope narrationEnvelope
	if err := sonic.UnmarshalString(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode narration: %w", err)
	}

	narrations := make(map[string]narratedFinding, len(envelope.Insights))
	for _, item := range envelope.Insights {
		if item.DedupeKey == "" {
			continue
		}
		narrations[item.DedupeKey] = item
	}

	return narrations, nil
}

type narrationEnvelope struct {
	Insights []narratedFinding `json:"insights"`
}

type narratedFinding struct {
	DedupeKey      string `json:"dedupeKey"`
	Headline       string `json:"headline"`
	Narrative      string `json:"narrative"`
	Recommendation string `json:"recommendation"`
}
