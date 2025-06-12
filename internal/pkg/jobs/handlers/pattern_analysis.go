package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	dlservice "github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// PatternAnalysisHandlerParams defines dependencies for the pattern analysis job handler
type PatternAnalysisHandlerParams struct {
	fx.In

	Logger            *logger.Logger
	SuggestionService *dlservice.SuggestionService
}

// PatternAnalysisHandler handles background jobs for analyzing shipment patterns
type PatternAnalysisHandler struct {
	logger            *zerolog.Logger
	suggestionService *dlservice.SuggestionService
}

// NewPatternAnalysisHandler creates a new pattern analysis job handler
func NewPatternAnalysisHandler(p PatternAnalysisHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "pattern_analysis").
		Logger()

	return &PatternAnalysisHandler{
		logger:            &log,
		suggestionService: p.SuggestionService,
	}
}

// JobType returns the job type this handler processes
func (pah *PatternAnalysisHandler) JobType() jobs.JobType {
	return jobs.JobTypeAnalyzePatterns
}

// ProcessTask processes a pattern analysis job
//
//nolint:funlen // this is a long function, but it's okay.
func (pah *PatternAnalysisHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	jobStartTime := time.Now()

	log := pah.logger.With().
		Str("job_id", task.ResultWriter().TaskID()).
		Str("job_type", task.Type()).
		Time("job_started_at", jobStartTime).
		Logger()

	log.Info().Msg("starting pattern analysis job")

	log.Info().Interface("payload", task.Payload()).Msg("payload")

	// Unmarshal job payload
	var payload jobs.PatternAnalysisPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().
			Err(err).
			Dur("job_duration", time.Since(jobStartTime)).
			Msg("failed to unmarshal pattern analysis payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	// Enhanced logging with more context
	logEntry := log.Info().
		Str("org_id", payload.OrganizationID.String()).
		Str("business_unit_id", payload.BusinessUnitID.String()).
		Int64("start_date", payload.StartDate).
		Int64("end_date", payload.EndDate).
		Int64("date_range_days", (payload.EndDate-payload.StartDate)/86400).
		Int64("min_frequency", payload.MinFrequency).
		Str("trigger_reason", payload.TriggerReason)

	if payload.CustomerID != nil {
		logEntry = logEntry.Str("customer_id", payload.CustomerID.String())
	}

	logEntry.Msg("processing pattern analysis with parameters")

	// Build pattern analysis request
	analysisReq := &dedicatedlane.PatternAnalysisRequest{
		StartDate:       payload.StartDate,
		EndDate:         payload.EndDate,
		CustomerID:      payload.CustomerID,
		ExcludeExisting: true, // Don't suggest lanes that already exist
		Config: &dedicatedlane.PatternDetectionConfig{
			MinFrequency:          payload.MinFrequency,
			AnalysisWindowDays:    (payload.EndDate - payload.StartDate) / 86400, // Convert to days
			MinConfidenceScore:    dedicatedlane.DefaultPatternDetectionConfig().MinConfidenceScore,
			SuggestionTTLDays:     dedicatedlane.DefaultPatternDetectionConfig().SuggestionTTLDays,
			RequireExactMatch:     false, // More lenient for automatic analysis
			WeightRecentShipments: true,
		},
	}

	// Override with custom frequency if provided, otherwise use default
	if payload.MinFrequency <= 0 {
		analysisReq.Config.MinFrequency = dedicatedlane.DefaultPatternDetectionConfig().MinFrequency
	}

	// Perform pattern analysis with detailed logging
	analysisStartTime := time.Now()
	log.Info().Msg("starting pattern analysis execution")

	result, err := pah.suggestionService.AnalyzePatterns(ctx, analysisReq)
	if err != nil {
		log.Error().
			Err(err).
			Dur("analysis_duration", time.Since(analysisStartTime)).
			Dur("total_job_duration", time.Since(jobStartTime)).
			Msg("pattern analysis failed")
		return fmt.Errorf("analyze patterns: %w", err)
	}

	// Comprehensive result logging with performance metrics
	log.Info().
		Int64("total_patterns_detected", result.TotalPatternsDetected).
		Int64("patterns_above_threshold", result.PatternsAboveThreshold).
		Int64("suggestions_created", result.SuggestionsCreated).
		Int64("suggestions_skipped", result.SuggestionsSkipped).
		Int64("analysis_processing_time_ms", result.ProcessingTimeMs).
		Dur("analysis_duration", time.Since(analysisStartTime)).
		Dur("total_job_duration", time.Since(jobStartTime)).
		Str("trigger_reason", payload.TriggerReason).
		Float64("pattern_qualification_rate", func() float64 {
			if result.TotalPatternsDetected > 0 {
				return float64(
					result.PatternsAboveThreshold,
				) / float64(
					result.TotalPatternsDetected,
				) * 100
			}
			return 0
		}()).
		Float64("suggestion_success_rate", func() float64 {
			total := result.SuggestionsCreated + result.SuggestionsSkipped
			if total > 0 {
				return float64(result.SuggestionsCreated) / float64(total) * 100
			}
			return 0
		}()).
		Msg("pattern analysis completed successfully")

	// Store result metadata in task result (optional, for monitoring)
	resultData := map[string]any{
		"total_patterns":      result.TotalPatternsDetected,
		"qualified_patterns":  result.PatternsAboveThreshold,
		"suggestions_created": result.SuggestionsCreated,
		"suggestions_skipped": result.SuggestionsSkipped,
		"processing_time_ms":  result.ProcessingTimeMs,
		"analysis_start_date": result.AnalysisStartDate,
		"analysis_end_date":   result.AnalysisEndDate,
		"trigger_reason":      payload.TriggerReason,
	}

	if writer := task.ResultWriter(); writer != nil {
		if data, dErr := jobs.MarshalPayload(resultData); dErr == nil {
			_, _ = writer.Write(data)
		}
	}

	return nil
}

// ExpireSuggestionsHandlerParams defines dependencies for the expire suggestions job handler
type ExpireSuggestionsHandlerParams struct {
	fx.In

	Logger            *logger.Logger
	SuggestionService *dlservice.SuggestionService
}

// ExpireSuggestionsHandler handles background jobs for expiring old suggestions
type ExpireSuggestionsHandler struct {
	logger            *zerolog.Logger
	suggestionService *dlservice.SuggestionService
}

// NewExpireSuggestionsHandler creates a new expire suggestions job handler
func NewExpireSuggestionsHandler(p ExpireSuggestionsHandlerParams) jobs.JobHandler {
	log := p.Logger.With().
		Str("handler", "expire_suggestions").
		Logger()

	return &ExpireSuggestionsHandler{
		logger:            &log,
		suggestionService: p.SuggestionService,
	}
}

// JobType returns the job type this handler processes
func (esh *ExpireSuggestionsHandler) JobType() jobs.JobType {
	return jobs.JobTypeExpireOldSuggestions
}

// ProcessTask processes an expire suggestions job
func (esh *ExpireSuggestionsHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	jobStartTime := time.Now()

	log := esh.logger.With().
		Str("job_id", task.ResultWriter().TaskID()).
		Str("job_type", task.Type()).
		Time("job_started_at", jobStartTime).
		Logger()

	log.Info().Msg("starting expire old suggestions job")

	// Unmarshal job payload
	var payload jobs.ExpireSuggestionsPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().
			Err(err).
			Dur("job_duration", time.Since(jobStartTime)).
			Msg("failed to unmarshal expire suggestions payload")
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Info().
		Str("org_id", payload.OrganizationID.String()).
		Str("business_unit_id", payload.BusinessUnitID.String()).
		Int("batch_size", payload.BatchSize).
		Msg("processing expire suggestions")

	// Execute expiration with timing
	expirationStartTime := time.Now()
	log.Info().Msg("starting suggestion expiration process")

	expiredCount, err := esh.suggestionService.ExpireOldSuggestions(
		ctx,
		payload.OrganizationID,
		payload.BusinessUnitID,
	)
	if err != nil {
		log.Error().
			Err(err).
			Dur("expiration_duration", time.Since(expirationStartTime)).
			Dur("total_job_duration", time.Since(jobStartTime)).
			Msg("expire suggestions failed")
		return fmt.Errorf("expire old suggestions: %w", err)
	}

	log.Info().
		Int64("expired_count", expiredCount).
		Dur("expiration_duration", time.Since(expirationStartTime)).
		Dur("total_job_duration", time.Since(jobStartTime)).
		Msg("expire old suggestions completed successfully")

	// Store result in task result
	resultData := map[string]any{
		"expired_count": expiredCount,
	}

	if writer := task.ResultWriter(); writer != nil {
		if data, dErr := jobs.MarshalPayload(resultData); dErr == nil {
			_, _ = writer.Write(data)
		}
	}

	return nil
}
