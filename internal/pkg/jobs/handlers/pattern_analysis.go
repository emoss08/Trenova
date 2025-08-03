/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
	"github.com/samber/oops"
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
func (pah *PatternAnalysisHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	jobStartTime := time.Now()

	log := pah.logger.With().
		Str("job_id", task.ResultWriter().TaskID()).
		Str("job_type", task.Type()).
		Time("job_started_at", jobStartTime).
		Logger()

	log.Info().Msg("starting pattern analysis job")

	// * Unmarshal job payload
	var payload jobs.PatternAnalysisPayload
	if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
		log.Error().
			Err(err).
			Dur("job_duration", time.Since(jobStartTime)).
			Msg("failed to unmarshal pattern analysis payload")
		return oops.In("pattern_analysis_handler").
			With("op", "process_task").
			With("job_id", task.ResultWriter().TaskID()).
			With("job_type", task.Type()).
			With("job_started_at", jobStartTime).
			Time(time.Now()).
			Wrapf(err, "unmarshal pattern analysis payload")
	}

	// * Build pattern analysis request
	analysisReq := &dedicatedlane.PatternAnalysisRequest{
		ExcludeExisting: true, // * Don't suggest lanes that already exist
		Config: &dedicatedlane.PatternDetectionConfig{
			MinFrequency:          payload.MinFrequency,
			MinConfidenceScore:    dedicatedlane.DefaultPatternDetectionConfig().MinConfidenceScore,
			SuggestionTTLDays:     dedicatedlane.DefaultPatternDetectionConfig().SuggestionTTLDays,
			RequireExactMatch:     false, // More lenient for automatic analysis
			WeightRecentShipments: true,
		},
	}

	// * Override with custom frequency if provided, otherwise use default
	if payload.MinFrequency <= 0 {
		analysisReq.Config.MinFrequency = dedicatedlane.DefaultPatternDetectionConfig().MinFrequency
	}

	// * Perform pattern analysis with detailed logging
	analysisStartTime := time.Now()
	log.Info().Msg("starting pattern analysis execution")

	result, err := pah.suggestionService.AnalyzePatterns(ctx, analysisReq)
	if err != nil {
		log.Error().
			Err(err).
			Dur("analysis_duration", time.Since(analysisStartTime)).
			Dur("total_job_duration", time.Since(jobStartTime)).
			Msg("pattern analysis failed")
		return oops.In("pattern_analysis_handler").
			With("op", "process_task").
			With("job_id", task.ResultWriter().TaskID()).
			With("job_type", task.Type()).
			With("job_started_at", jobStartTime).
			Time(time.Now()).
			Wrapf(err, "analyze patterns")
	}

	// * Store result metadata in task result (optional, for monitoring)
	resultData := map[string]any{
		"total_patterns":     result.TotalPatternsDetected,
		"qualified_patterns": result.PatternsAboveThreshold,
		"processing_time_ms": result.ProcessingTimeMs,
		"trigger_reason":     payload.TriggerReason,
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
