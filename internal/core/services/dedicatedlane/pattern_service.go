package dedicatedlane

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type PatternServiceParams struct {
	fx.In

	Logger            *logger.Logger
	ShipmentRepo      repositories.ShipmentRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	SuggestionRepo    repositories.DedicatedLaneSuggestionRepository
}

type PatternService struct {
	l            *zerolog.Logger
	shipmentRepo repositories.ShipmentRepository
	dlRepo       repositories.DedicatedLaneRepository
	suggRepo     repositories.DedicatedLaneSuggestionRepository
}

func NewPatternService(p PatternServiceParams) *PatternService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_pattern").
		Logger()

	return &PatternService{
		l:            &log,
		shipmentRepo: p.ShipmentRepo,
		dlRepo:       p.DedicatedLaneRepo,
		suggRepo:     p.SuggestionRepo,
	}
}

// AnalyzePatterns performs pattern analysis and creates suggestions
func (ps *PatternService) AnalyzePatterns(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
) (*dedicatedlane.PatternAnalysisResult, error) {
	startTime := time.Now()

	log := ps.l.With().
		Str("operation", "AnalyzePatterns").
		Str("orgId", req.OrganizationID.String()).
		Logger()

	// Use default config if none provided
	config := req.Config
	if config == nil {
		config = dedicatedlane.DefaultPatternDetectionConfig()
	}

	log.Info().Msg("starting pattern analysis")

	// Get shipments for analysis
	shipments, err := ps.getShipmentsForAnalysis(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipments for analysis")
		return nil, err
	}

	log.Info().Int("shipmentCount", len(shipments)).Msg("analyzing shipments")

	// Group shipments by pattern
	patterns := ps.groupShipmentsByPattern(shipments, config)

	log.Info().Int("patternCount", len(patterns)).Msg("patterns detected")

	// Filter patterns by frequency and confidence
	qualifiedPatterns := ps.filterPatterns(patterns, config)

	log.Info().Int("qualifiedPatterns", len(qualifiedPatterns)).Msg("patterns above threshold")

	// Check for existing dedicated lanes and suggestions
	if req.ExcludeExisting {
		qualifiedPatterns = ps.excludeExistingLanes(
			ctx,
			qualifiedPatterns,
			req.OrganizationID,
			req.BusinessUnitID,
		)
	}

	// Create suggestions
	suggestionsCreated := int64(0)
	suggestionsSkipped := int64(0)

	for _, pattern := range qualifiedPatterns {
		suggestion := ps.createSuggestionFromPattern(pattern, req, config)

		_, err = ps.suggRepo.Create(ctx, suggestion)
		if err != nil {
			log.Error().Err(err).Msg("failed to save suggestion")
			suggestionsSkipped++
			continue
		}

		suggestionsCreated++
	}

	result := &dedicatedlane.PatternAnalysisResult{
		TotalPatternsDetected:  int64(len(patterns)),
		PatternsAboveThreshold: int64(len(qualifiedPatterns)),
		SuggestionsCreated:     suggestionsCreated,
		SuggestionsSkipped:     suggestionsSkipped,
		AnalysisStartDate:      req.StartDate,
		AnalysisEndDate:        req.EndDate,
		ConfigUsed:             config,
		Patterns:               qualifiedPatterns,
		ProcessingTimeMs:       time.Since(startTime).Milliseconds(),
	}

	log.Info().
		Int64("suggestionsCreated", suggestionsCreated).
		Int64("processingTimeMs", result.ProcessingTimeMs).
		Msg("pattern analysis completed")

	return result, nil
}

func (ps *PatternService) getShipmentsForAnalysis(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
) ([]*shipment.Shipment, error) {
	// Build filter for completed/billed shipments within the date range
	filter := &repositories.ListShipmentOptions{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  req.OrganizationID,
				BuID:   req.BusinessUnitID,
				UserID: pulid.MustNew("usr_"), // TODO: Get from context
			},
			Limit:  10000, // Large limit for analysis
			Offset: 0,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			Status: fmt.Sprintf("%s,%s",
				string(shipment.StatusCompleted),
				string(shipment.StatusBilled)),
		},
	}

	// TODO: Add date range filtering to repository if not already available
	result, err := ps.shipmentRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter by date range and customer if specified
	var filteredShipments []*shipment.Shipment
	for _, shp := range result.Items {
		// Check date range
		if shp.CreatedAt < req.StartDate || shp.CreatedAt > req.EndDate {
			continue
		}

		// Filter by customer if specified
		if req.CustomerID != nil && !pulid.Equals(shp.CustomerID, *req.CustomerID) {
			continue
		}

		filteredShipments = append(filteredShipments, shp)
	}

	return filteredShipments, nil
}

func (ps *PatternService) groupShipmentsByPattern(
	shipments []*shipment.Shipment,
	config *dedicatedlane.PatternDetectionConfig,
) []*dedicatedlane.PatternMatch {
	patternMap := make(map[string]*dedicatedlane.PatternMatch)

	for _, shp := range shipments {
		// Skip shipments without required data
		if shp.CustomerID.IsNil() || shp.Moves == nil || len(shp.Moves) == 0 {
			continue
		}

		// Get origin and destination from first and last moves
		var originLocationID, destLocationID pulid.ID
		if len(shp.Moves) > 0 && len(shp.Moves[0].Stops) > 0 {
			originLocationID = shp.Moves[0].Stops[0].LocationID
		}
		if len(shp.Moves) > 0 && len(shp.Moves[len(shp.Moves)-1].Stops) > 0 {
			lastMove := shp.Moves[len(shp.Moves)-1]
			destLocationID = lastMove.Stops[len(lastMove.Stops)-1].LocationID
		}

		if originLocationID.IsNil() || destLocationID.IsNil() {
			continue
		}

		// Create pattern key
		key := ps.createPatternKey(shp, originLocationID, destLocationID, config)

		pattern, exists := patternMap[key]
		if !exists {
			pattern = &dedicatedlane.PatternMatch{
				CustomerID:            shp.CustomerID,
				OriginLocationID:      originLocationID,
				DestinationLocationID: destLocationID,
				ServiceTypeID:         &shp.ServiceTypeID,
				ShipmentTypeID:        &shp.ShipmentTypeID,
				TrailerTypeID:         shp.TrailerTypeID,
				TractorTypeID:         shp.TractorTypeID,
				FrequencyCount:        0,
				ShipmentIDs:           make([]pulid.ID, 0),
				PatternDetails:        make(map[string]any),
				AverageFreightCharge:  &decimal.NullDecimal{},
				TotalFreightValue:     &decimal.NullDecimal{},
			}
			patternMap[key] = pattern
		}

		// Update pattern metrics
		pattern.FrequencyCount++
		pattern.ShipmentIDs = append(pattern.ShipmentIDs, shp.ID)

		// Track date range
		if pattern.FirstShipmentDate == 0 || shp.CreatedAt < pattern.FirstShipmentDate {
			pattern.FirstShipmentDate = shp.CreatedAt
		}
		if shp.CreatedAt > pattern.LastShipmentDate {
			pattern.LastShipmentDate = shp.CreatedAt
		}

		// Update freight charges
		if shp.FreightChargeAmount.Valid {
			if !pattern.TotalFreightValue.Valid {
				pattern.TotalFreightValue = &decimal.NullDecimal{
					Decimal: shp.FreightChargeAmount.Decimal,
					Valid:   true,
				}
			} else {
				pattern.TotalFreightValue.Decimal = pattern.TotalFreightValue.Decimal.Add(shp.FreightChargeAmount.Decimal)
			}
		}
	}

	// Convert map to slice and calculate metrics
	patterns := make([]*dedicatedlane.PatternMatch, 0, len(patternMap))
	for _, pattern := range patternMap {
		// Calculate average freight charge
		if pattern.TotalFreightValue.Valid && pattern.FrequencyCount > 0 {
			avgCharge := pattern.TotalFreightValue.Decimal.Div(
				decimal.NewFromInt(pattern.FrequencyCount),
			)
			pattern.AverageFreightCharge = &decimal.NullDecimal{
				Decimal: avgCharge,
				Valid:   true,
			}
		}

		// Calculate confidence score
		pattern.ConfidenceScore = ps.calculateConfidenceScore(pattern, config)

		patterns = append(patterns, pattern)
	}

	return patterns
}

func (ps *PatternService) createPatternKey(
	shp *shipment.Shipment,
	originID, destID pulid.ID,
	config *dedicatedlane.PatternDetectionConfig,
) string {
	key := fmt.Sprintf("%s|%s|%s",
		shp.CustomerID.String(),
		originID.String(),
		destID.String())

	// Include equipment/service types if exact match required
	if config.RequireExactMatch {
		if shp.ServiceTypeID != pulid.Nil {
			key += "|" + shp.ServiceTypeID.String()
		}
		if shp.ShipmentTypeID != pulid.Nil {
			key += "|" + shp.ShipmentTypeID.String()
		}
		if shp.TrailerTypeID != nil && !shp.TrailerTypeID.IsNil() {
			key += "|" + shp.TrailerTypeID.String()
		}
		if shp.TractorTypeID != nil && !shp.TractorTypeID.IsNil() {
			key += "|" + shp.TractorTypeID.String()
		}
	}

	return key
}

func (ps *PatternService) calculateConfidenceScore(
	pattern *dedicatedlane.PatternMatch,
	config *dedicatedlane.PatternDetectionConfig,
) decimal.Decimal {
	score := decimal.NewFromFloat(0.0)

	// Base score from frequency (normalized to 0-0.4)
	frequencyScore := decimal.NewFromInt(pattern.FrequencyCount).Div(decimal.NewFromInt(10))
	if frequencyScore.GreaterThan(decimal.NewFromFloat(0.4)) {
		frequencyScore = decimal.NewFromFloat(0.4)
	}
	score = score.Add(frequencyScore)

	// Recency bonus (0-0.3)
	if config.WeightRecentShipments {
		daysSinceLastShipment := (timeutils.NowUnix() - pattern.LastShipmentDate) / 86400
		switch {
		case daysSinceLastShipment <= 7:
			score = score.Add(decimal.NewFromFloat(0.3))
		case daysSinceLastShipment <= 30:
			score = score.Add(decimal.NewFromFloat(0.2))
		case daysSinceLastShipment <= 60:
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// Consistency bonus (0-0.2)
	timeSpan := pattern.LastShipmentDate - pattern.FirstShipmentDate
	if timeSpan > 0 {
		avgDaysBetween := timeSpan / (86400 * (pattern.FrequencyCount - 1))
		if avgDaysBetween <= 30 { // Regular monthly pattern
			score = score.Add(decimal.NewFromFloat(0.2))
		} else if avgDaysBetween <= 60 {
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// Value bonus (0-0.1)
	if pattern.TotalFreightValue.Valid &&
		pattern.TotalFreightValue.Decimal.GreaterThan(decimal.NewFromFloat(10000)) {
		score = score.Add(decimal.NewFromFloat(0.1))
	}

	// Cap at 1.0
	if score.GreaterThan(decimal.NewFromFloat(1.0)) {
		score = decimal.NewFromFloat(1.0)
	}

	return score
}

func (ps *PatternService) filterPatterns(
	patterns []*dedicatedlane.PatternMatch,
	config *dedicatedlane.PatternDetectionConfig,
) []*dedicatedlane.PatternMatch {
	var filtered []*dedicatedlane.PatternMatch

	for _, pattern := range patterns {
		if pattern.FrequencyCount >= config.MinFrequency &&
			pattern.ConfidenceScore.GreaterThanOrEqual(config.MinConfidenceScore) {
			filtered = append(filtered, pattern)
		}
	}

	// Sort by confidence score descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ConfidenceScore.GreaterThan(filtered[j].ConfidenceScore)
	})

	return filtered
}

func (ps *PatternService) excludeExistingLanes(
	_ context.Context,
	patterns []*dedicatedlane.PatternMatch,
	_, _ pulid.ID,
) []*dedicatedlane.PatternMatch {
	// TODO: Implement check against existing dedicated lanes
	// This would require querying the dedicated lanes repository
	// to see if a lane already exists for the same route/customer
	return patterns
}

func (ps *PatternService) createSuggestionFromPattern(
	pattern *dedicatedlane.PatternMatch,
	req *dedicatedlane.PatternAnalysisRequest,
	config *dedicatedlane.PatternDetectionConfig,
) *dedicatedlane.DedicatedLaneSuggestion {
	now := timeutils.NowUnix()
	expiresAt := now + (config.SuggestionTTLDays * 86400)

	// Generate suggested name
	suggestedName := ps.generateSuggestedName(pattern)

	suggestion := &dedicatedlane.DedicatedLaneSuggestion{
		BusinessUnitID:        req.BusinessUnitID,
		OrganizationID:        req.OrganizationID,
		Status:                dedicatedlane.SuggestionStatusPending,
		CustomerID:            pattern.CustomerID,
		OriginLocationID:      pattern.OriginLocationID,
		DestinationLocationID: pattern.DestinationLocationID,
		ServiceTypeID:         pattern.ServiceTypeID,
		ShipmentTypeID:        pattern.ShipmentTypeID,
		TrailerTypeID:         pattern.TrailerTypeID,
		TractorTypeID:         pattern.TractorTypeID,
		ConfidenceScore:       pattern.ConfidenceScore,
		FrequencyCount:        pattern.FrequencyCount,
		AverageFreightCharge:  pattern.AverageFreightCharge,
		TotalFreightValue:     pattern.TotalFreightValue,
		LastShipmentDate:      pattern.LastShipmentDate,
		FirstShipmentDate:     pattern.FirstShipmentDate,
		AnalysisStartDate:     req.StartDate,
		AnalysisEndDate:       req.EndDate,
		SuggestedName:         suggestedName,
		PatternDetails: map[string]any{
			"shipmentIds":    pattern.ShipmentIDs,
			"analysisConfig": config,
			"detectionTime":  now,
		},
		ExpiresAt: expiresAt,
	}

	return suggestion
}

func (ps *PatternService) generateSuggestedName(pattern *dedicatedlane.PatternMatch) string {
	// This would ideally use location names, but we'd need to join with location data
	// For now, use a simple format with IDs
	return fmt.Sprintf("Lane-%s-to-%s",
		pattern.OriginLocationID.String()[:8],
		pattern.DestinationLocationID.String()[:8])
}
